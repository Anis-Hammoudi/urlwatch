# Documentation Architecturale (URLWatch)

## 1. Découpage et Architecture
Le projet suit une architecture hexagonale (ou Clean Architecture) adaptée à Go, en utilisant le dossier `internal/` pour encapsuler le code métier :
* **`internal/domain`** : C'est le cœur du système. J'y ai placé les structures métier (`Batch`, `CheckResult`) et, surtout, les interfaces `Checker` et `Store`. L'inversion de dépendance est primordiale ici : ce sont les consommateurs (le pool, l'API) qui définissent les interfaces dont ils ont besoin, rendant les implémentations (mémoire, HTTP) interchangeables.
* **`internal/api`** : La couche de transport. Les DTOs (Data Transfer Objects) y sont séparés des modèles du domaine pour s'assurer que des changements dans le contrat JSON n'impactent pas la logique métier.
* **`cmd/urlwatch/main.go`** : Reste le plus fin possible. Il se contente d'initialiser les dépendances (logger, store, checker), de les injecter, et de gérer le cycle de vie du serveur (arrêt gracieux).

## 2. Modèle de Concurrence
J'ai opté pour le pattern **Fan-out / Fan-in** :
* **Canaux non bufferisés** : J'ai délibérément utilisé des channels non bufferisés (`jobs := make(chan string)`). Cela crée une "backpressure" mécanique (synchronisation stricte). Le producteur ne peut envoyer une URL que si un worker est explicitement prêt à la traiter. Cela évite d'allouer inutilement de la mémoire pour stocker des milliers d'URLs en attente dans un buffer si la requête est annulée.
* **Gestion des échecs partiels** : Si une URL échoue (ex: erreur DNS), le worker encapsule l'erreur dans la structure `CheckResult` sans faire paniquer le pool. Le collecteur (Fan-in) agrège ces résultats séquentiellement (`up++`, `down++`). Comme le collecteur tourne dans une seule goroutine, il n'y a **aucune data race** lors de l'agrégation (confirmé par `go test -race`), rendant l'usage de `sync.Mutex` inutile ici.

## 3. Prévention des fuites de Goroutines (Goroutine Leaks)
Le design élimine les risques de fuites par deux mécanismes stricts :
1.  **Propagation du `context.Context`** : Si le client annule sa requête HTTP ou si le timeout (défini dans les options) est atteint, `ctx.Done()` est déclenché. La goroutine productrice s'arrête instantanément et ferme le channel `jobs`.
2.  **Cascade de fermeture via `sync.WaitGroup`** : La fermeture de `jobs` force les workers à sortir de leur boucle `range`. Le `WaitGroup` atteint 0, ce qui déclenche la fermeture du channel `results`, libérant ainsi la goroutine principale.

## 4. Stratégie d'Erreurs
Les erreurs sont considérées comme des valeurs (philosophie Go) :
* **Erreurs Sentinelles** : `ErrBatchNotFound` est définie dans le `domain`. La couche API utilise `errors.Is()` pour traduire cette erreur métier précise en un code HTTP `404 Not Found`.
* **Wrapping** : Les erreurs remontées (comme l'échec d'une vérification ou du JSON) conservent leur contexte grâce aux types personnalisés ou au wrapping pour un débuggage facilité, tout en renvoyant le format JSON strict exigé par le contrat.

## 5. Philosophie Go vs Autres Langages
Pourquoi Go est un excellent choix ici :
1.  **Concurrence native et légère** : Lancer 50 workers via des goroutines consomme infiniment moins de mémoire (quelques KB) que de lancer 50 threads OS en Java ou en C++.
2.  **Interfaces implicites** : Pas besoin de `implements Checker` comme en Java. Cela m'a permis de créer facilement un `mockChecker` directement dans mes fichiers de tests sans polluer le code de production.
3.  **JSON Struct Tags** : La sérialisation JSON via les tags (`json:"status_code,omitempty"`) permet de respecter le contrat d'API au caractère près sans avoir à écrire de sérialiseurs complexes comme en Python.
* **Limite ressentie** : L'absence de types algébriques de données (Enums stricts) ou de pattern matching exhaustif (comme en Rust) rend la validation de certains états d'erreurs légèrement plus verbeuse (répétition des `if err != nil`).