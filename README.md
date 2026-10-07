# SaaS Starterkit - Simple SaaS Starterkit with Next.js, DrizzleORM, and PostgreSQL

## Features

- **Next.js**: Full-stack React framework for building server-rendered applications.
- **DrizzleORM**: Type-safe ORM for PostgreSQL, providing a simple and efficient way to interact with the database. Migration is done by `instrumentation.ts` file.
- **PostgreSQL**: Reliable and powerful open-source relational database management system.
- **Docker**: Containerization for easy deployment and environment consistency.
- **Better Auth**: Authentication system for managing user access and permissions.
- **Email Notifications**: Built-in email client for sending notifications and alerts.
- **Tailwind CSS**: Utility-first CSS framework for rapid UI development.
- **Background Tasks**: Support for running background tasks like CRON jobs or Kafka consumers.

## Getting Started

Run the `bun run clear` to clear the migrations folder and the database.
After configuring the `schema.ts` file, generate new migrations with:

```bash
bun run create:migration
```

## Local Development

```bash
cp .env.example .env # Copy the example environment variables
bun run local:up # Runs the local PostgreSQL database
bun run dev # Starts the Next.js development server
```

### 한 번에 실행 (DB + Ollama + AI 서버 + 앱)

```bash
make run              # Ctrl+C로 한꺼번에 종료. 앱 http://localhost:3000, AI 서버 :8000
make run LLM=mock     # Ollama 없이 키워드 분류기로 (UI/연동 확인용)
make run PORT=3100    # 포트 변경 (BETTER_AUTH_URL도 같이 맞춰 준다)
make status           # 무엇이 떠 있는지
make stop             # 앱/AI 서버 포트의 리스너 종료 (DB, Ollama는 그대로)
make stop-ollama      # 모델을 내리고 ollama serve 종료
make check            # 타입 체크 + 린트 + AI 서버 테스트
```

처음 한 번은 `.env`(`cp .env.example .env`)가 필요하고, 로컬 모델은 `ollama pull gemma4:12b`로 받아 둔다. 의존성(`npm install`, `uv sync`)은 없을 때 `make run`이 설치한다. 로그인 OTP는 SMTP를 설정하지 않으면 `[next]` 로그에 출력된다. AI 서버 자세한 내용은 [`ai-service/README.md`](./ai-service/README.md).

## Quick Links

- [Next.js Documentation](https://nextjs.org/docs)
- [DrizzleORM Documentation](https://orm.drizzle.team/)
- [Better Auth Documentation](https://better-auth.com/docs)
