# URLWatch

URLWatch est un microservice léger et performant écrit en Go 1.22. Il permet la vérification d'URLs en masse de manière concurrente. Les clients peuvent soumettre un lot d'URLs, définir le niveau de parallélisme (worker pool) et un délai d'expiration (timeout), puis consulter les résultats agrégés a posteriori.

## Prérequis

* **Go 1.22** ou supérieur (utilise les nouvelles fonctionnalités de routage de `net/http` et `log/slog`).

## Compiler et Exécuter

Le projet utilise les modules Go standards et ne requiert aucune dépendance externe tierce.

**1. Télécharger les dépendances (le cas échéant)**
```bash
go mod tidy
```

**2. Compiler le binaire**
```bash
go build -o urlwatch ./cmd/urlwatch
```

**3. Exécuter le service**
```bash
go run ./cmd/urlwatch/main.go
```
Le serveur démarrera sur le port 8080 par défaut.

## Tests et Qualité du code
L'ensemble de la logique métier (Worker Pool, API, Store) est couvert par des tests. Le moteur concurrent a été testé contre les accès concurrents non sécurisés (data races).

Exécuter tous les tests avec le détecteur de race conditions :
```bash
go test ./... -v -race
```

Vérifier la propreté du code (linter standard) :
```bash
go vet ./...
```

## Exemples d'utilisation de l'API (cURL)
Voici comment interagir avec l'API REST du microservice.

### 1. Sonde de vivacité (Healthcheck)
Permet de vérifier que le service est en ligne (ne pollue pas les logs applicatifs).

Requête :
```bash
curl http://localhost:8080/healthz
```

Réponse attendue (200 OK) :
```plaintext
ok
```

### 2. Créer un lot de vérifications (POST)
Soumet une liste d'URLs à vérifier. Le service effectue les requêtes en parallèle via un Worker Pool borné.

Requête :
```bash
curl -X POST http://localhost:8080/v1/checks \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://go.dev", 
      "https://google.com", 
      "https://ceci-est-une-url-invalide.invalid"
    ], 
    "options": {
      "concurrency": 2, 
      "timeout_ms": 5000
    }
  }'
```

Réponse attendue (201 Created) :
```json
{
  "batch_id": "b_a1b2c3",
  "created_at": "2026-06-19T10:00:00Z",
  "summary": {
    "total": 3,
    "up": 2,
    "down": 1,
    "duration_ms": 312
  },
  "results": [
    {
      "url": "https://go.dev",
      "status_code": 200,
      "ok": true,
      "latency_ms": 120
    },
    {
      "url": "https://google.com",
      "status_code": 200,
      "ok": true,
      "latency_ms": 145
    },
    {
      "url": "https://ceci-est-une-url-invalide.invalid",
      "ok": false,
      "latency_ms": 47,
      "error": "Get \"https://ceci-est-une-url-invalide.invalid\": dial tcp: lookup ceci-est-une-url-invalide.invalid: no such host"
    }
  ]
}
```

### 3. Consulter un lot existant (GET)
Récupère les résultats d'un lot précédemment calculé à l'aide de son identifiant (batch_id).

Requête :
```bash
curl http://localhost:8080/v1/checks/b_a1b2c3
```

*(Renvoie le même objet JSON que ci-dessus, avec un code 200 OK, ou un code 404 Not Found si l'identifiant n'existe pas dans la base en mémoire).*

## Architecture du projet

* **cmd/urlwatch/** : Point d'entrée de l'application. Assemble les dépendances et gère l'arrêt gracieux (Graceful Shutdown).
* **internal/domain/** : Cœur métier. Contient les entités (Batch, CheckResult) et les interfaces abstraites (Checker, Store).
* **internal/pool/** : Moteur de concurrence implémentant le pattern Fan-out / Fan-in avec gestion stricte de l'annulation par context.
* **internal/checker/** : Implémentation du client HTTP réel et du mock déterministe pour les tests.
* **internal/store/** : Stockage en mémoire map protégé par sync.RWMutex contre les data races.
* **internal/api/** : Couche HTTP (Routeur net/http natif Go 1.22, Middlewares de logs slog et Recovery, DTOs JSON).
