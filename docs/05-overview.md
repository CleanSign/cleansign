# 5. Overview (Wrap-up)

시스템 디자인 인터뷰 순서를 따라 진행한 설계 전체 요약입니다.

1. [`01-fr.md`](./01-fr.md) — 기능 요구사항, MoSCoW
2. [`02-api-ui-design.md`](./02-api-ui-design.md) — 화면 흐름 + REST API
3. [`03-high-level-design.md`](./03-high-level-design.md) — 시스템 경계 + 인프라 구조
4. [`04-data-schema-modeling.md`](./04-data-schema-modeling.md) — 테이블 스키마
5. [`06-ai-analysis-plan.md`](./06-ai-analysis-plan.md) — AI 계약서 분석 설계·개발 계획
6. [`07-go-server-migration-plan.md`](./07-go-server-migration-plan.md), [`08-go-server-changes.md`](./08-go-server-changes.md) — Go 서버 전환 계획과 결과

인터랙티브 캔버스:
- 화면 설계 + API: https://claude.ai/artifact/8WdxGbu3DPzr7r7tqGyFTk
- ERD(설계 당시 기준): https://claude.ai/artifact/BNsxk821QQ7qSEr93Lbum6
- 하이레벨 아키텍처(설계 당시 기준, 현재 구조와 다름): https://claude.ai/artifact/PMWpV9YT1qWXAdHPAqSaQF

---

## 단계별 핵심 요약

**FR** — Must는 온보딩/인증과 계약서 분석 핵심(업로드→분석→결과)뿐. 구독·결제·상담/예약은 전부 Should로, 소식·알림 설정·FAQ는 Could로 내려서 첫 릴리스는 "무료로 분석까지"에 집중.

**API + UI** — 홈이 허브인 구조. 계약서 분석은 처음에는 `POST /contracts` 하나의 동기 응답으로 설계했지만, 실측 지연(조항당 수 초)으로 **비동기로 바꿨다**: `POST`는 202를 즉시 돌려주고 화면은 `GET /contracts/{id}`를 1.5초마다 폴링해 실제 처리 단계(`progress`)를 보여 준다(2026-10-06 결정 변경). 로그인은 이메일 인증코드 + JWT(server가 직접 구현). FAQ는 DB/API 없이 클라이언트 하드코딩.

**High Level Design** — 지금 구조는 **web(Next.js, 화면) → server(Go, API·로그인·분석 작업) → ai-service(Python, 분석 파이프라인)** 와 PostgreSQL, SMTP다(3-1). 확장 방향은 로드밸런서 + server ×N + **DB 기반 작업 큐 + 분석 워커**이고, 이전 판의 Kafka RPC는 더 커졌을 때의 선택지로 낮췄다(3-2). 클라이언트가 보는 API 계약(202 + 폴링)은 확장해도 그대로다. 나머지(구독/상담/결제)는 server → PG, server → 외부 PG 수준의 단순 흐름이라 별도 다이어그램 불필요.

**Data Schema** — 인증은 `users` · `sessions` · `email_codes`(Better Auth 테이블 대신 직접 설계), `user_consent`는 append-only 이력, `notification_settings`는 1:1 현재 상태(미구현), `contract.loss_amount`/`booking.amount`는 분석·예약 시점 스냅샷으로 캐시. `contract`에는 폴링을 위한 `status` · `progress` · `error_message`가 있다. FAQ는 테이블 없음.

---

## 구현 현황과 다음 단계 (2026-10)

**구현됨**: 로그인(이메일 인증코드 + JWT), 약관 동의, 계약서 업로드 → 비동기 분석 → 결과·조항 상세, 히스토리·삭제, web / server(Go) / ai-service 분리. 상세는 [`07-go-server-migration-plan.md`](./07-go-server-migration-plan.md)(계획과 결과), [`08-go-server-changes.md`](./08-go-server-changes.md)(그림으로 보기)를 보세요.

**다음 단계**

1. 점수·등급 정의 수정과 정상 조항 라벨 확보(06 Phase 0~1) — 이게 끝나야 점수를 화면에 노출할 수 있다.
2. 법령·판례 RAG(06 Phase 3, FR-2.5).
3. 분석 작업 큐와 워커(3-2) — 서버 재시작에도 작업이 사라지지 않게 한다. 오브젝트 스토리지는 이때 같이 정한다.
4. 소식, 전문가 상담·예약, 구독·결제, 알림 설정(FR-3~6). 모바일 앱도 이후 단계다.
5. 배포 전: IP별 레이트 리밋, 운영용 메일 서비스, 배포·롤백 절차.
