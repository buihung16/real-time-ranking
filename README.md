#   Real-time Ranking Microservice

This microservice handles real-time ranking for a video system. It allows updating video scores based on user interactions and retrieving top-ranked videos. [cite: 20, 21, 22]

##   Requirements

-   Go 1.22 or later
-   Docker
-   Docker Compose

##   Getting Started

1.  **Clone the repository:**

    ```bash
    git clone <repository_url>
    cd realtime-ranking
    ```

2.  **Run with Docker Compose (Local Development):**

    ```bash
    docker-compose up -d
    ```

    This will start PostgreSQL, Redis, Kafka, Zookeeper, and the microservice.

3.  **Access the API:**

    The API will be available at `http://localhost:8080`.

4.  **View Swagger UI:**

    The Swagger UI for API documentation will be available at `http://localhost:8080/swagger/index.html`.

##   Configuration

The following environment variables can be used to configure the microservice:

-   `POSTGRES_URL`: PostgreSQL connection string (default: `postgres://myuser:mypassword@postgres:5432/mydb`)
-   `REDIS_URL`: Redis connection string (default: `redis://redis:6379/0`)
-   `KAFKA_BROKERS`: Comma-separated list of Kafka brokers (default: `kafka:9092`)

These can be set either in your environment or in the `docker-compose.yaml` file.

##   API Endpoints

(See the Swagger UI for detailed documentation.)

- `POST /videos`: Create a new video.
- `PUT /videos/{id}`: Update a video.
- `POST /videos/{id}/view`: Record a video view.
- `POST /videos/{id}/like`: Record a video like.
- `POST /videos/{id}/comment`: Record a video comment.
- `POST /videos/{id}/share`: Record a video share.
- `POST /videos/{id}/watch`: Record video watch time.
- `GET /videos/top`: Get top-ranked videos.
- `GET /users/{userID}/videos/top`: Get top-ranked videos for a user.
- `POST /users/{userID}/preferences`: Update user preferences.

##   Notes

-   This README provides basic setup instructions. For production deployments, you'll need more robust configuration, security, and monitoring.
-   The `getVideoCategoryMap` function in `services/services.go` is a placeholder. You'll need to implement actual logic to retrieve video categories.