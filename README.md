# Mutabaah Telegram Bot

This is a telegram bot that essentially acts as a wrapper for a deed tracker(mutabaah) website. It is linked together through a database hosted by [Appwrite](https://appwrite.io).

```mermaid
graph LR
    %% Users
    UserA([Telegram User])
    UserB([Web Visitor])

    %% Frontend Platforms
    Bot[Telegram Bot]
    Web[Website Frontend]

    %% Backend Database
    subgraph Cloud Backend
        DB[(Appwrite Database)]
    end

    %% Connections
    UserA <-->|Interacts with| Bot
    UserB <-->|Browses| Web

    Bot <-->|Reads/Writes Data| DB
    Web <-->|Syncs Data| DB

    %% Styling
    style DB fill:#f96,stroke:#333,stroke-width:2px

```

The Appwrite users database looks something like this:
| $id | telegram_username | telegram_id | data  |
|-----|-------------------|-------------|-------|
| 1   | johndoe           | 123         | {...} |
| 2   | solarft           | 456         | {...} |
| 3   | big_bean_can      | 789         | {...} |
