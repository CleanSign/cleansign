# CleanSign 쿠버네티스 배포 (local: Lima VM + k3s)

맥 안의 Lima VM(Ubuntu + k3s)에 web, server, ai-service, postgres를 올리는 구성이다.
그림과 쿠버네티스 개념 설명은 [ARCHITECTURE.md](ARCHITECTURE.md)를 본다.
이 폴더는 앱 소스를 참조하지 않는다. 앱과의 접점은 이미지 이름(`cleansign/<서비스>`)과 환경변수 이름뿐이라, 나중에 통째로 다른 레포로 옮길 수 있다.

```
브라우저 ─ http://localhost ─▶ Traefik(Ingress) ─▶ web ─(/api/v1 rewrite)─▶ server ─▶ postgres (PVC)
                                                                              └────▶ ai-service (mock)
```

## 폴더 구조

```
deploy/
├── Makefile                      # 자주 쓰는 명령 (아래 표)
├── scripts/images.sh             # 이미지 빌드 + VM 의 k3s 에 직접 import (레지스트리 없음)
└── k8s/
    ├── base/                     # 환경과 무관한 공통 매니페스트
    │   ├── namespace.yaml        #   cleansign
    │   ├── postgres.yaml         #   StatefulSet + PVC(2Gi)
    │   ├── ai-service.yaml       #   Deployment + Service
    │   ├── server.yaml           #   Deployment + Service
    │   ├── web.yaml              #   Deployment + Service
    │   └── ingress.yaml          #   Traefik Ingress (/ → web)
    └── overlays/local/           # 이 맥의 Lima VM 용 (이미지 태그, Secret)
        ├── kustomization.yaml
        └── secrets.env.example   #   `make secrets` 가 secrets.env 를 만든다 (Git 제외)
```

## 처음 세팅

전제: Docker(Docker Desktop 등)가 실행 중이고, `limactl`, `kubectl`, `openssl`이 설치돼 있다.

```bash
# 1) Lima VM 만들기 (처음 한 번, Ubuntu 이미지를 받느라 몇 분 걸린다)
limactl create --name=k3s-1 --cpus=2 --memory=4 --disk=20 --tty=false template://k3s
limactl start k3s-1

# 2) 이미지 3개를 빌드해서 VM 에 넣기
cd deploy
scripts/images.sh

# 3) 배포 (secrets.env 가 없으면 무작위 값으로 만들어 준다)
make up
make status        # 파드가 전부 Running 인지 확인
```

`deploy/Makefile`은 기본으로 Lima VM의 kubeconfig(`~/.lima/k3s-1/copied-from-guest/kubeconfig.yaml`)를 쓴다. 다른 클러스터(Docker Desktop 등)가 기본 컨텍스트여도 이 Makefile은 영향을 받지 않는다. 다른 클러스터에 올리려면 `make up KUBECONFIG=~/.kube/config`처럼 지정한다.

## 접속

| 방법 | 주소 | 비고 |
|---|---|---|
| Ingress | http://localhost | 클러스터에 상시 열려 있다. 실서버와 같은 경로 |
| `make forward` | web `:3000`, server `:4000`, ai `:8000`, db `:5433` | 터미널을 켜 두는 동안만. 디버깅·DB 접속용. 포트는 `WEB_PORT=3100` 식으로 바꾼다 |

- 로그인 인증코드는 메일로 가지 않는다(SMTP 미설정). server 로그에 찍힌다: `kubectl -n cleansign logs deploy/server --tail=20 | grep -E '[0-9]{6}'` 또는 k9s에서 server 파드에 커서를 놓고 `l`.
- DB 접속: `psql -h localhost -p 5433 -U postgres -d cleansign` (비밀번호는 `k8s/overlays/local/secrets.env`의 `POSTGRES_PASSWORD`, `make forward` 실행 중일 때).

## 개발 → 배포 흐름

개발 중 빠른 확인은 저장소 루트의 `make run`(로컬 프로세스)으로 하고, 클러스터에서 확인할 때만 아래를 쓴다. 모든 명령은 `deploy/`에서 실행한다.

| 상황 | 명령 |
|---|---|
| 서버/AI/웹 코드를 고쳤다 | `make deploy S=server` (또는 `ai-service`, `web`) |
| DB 스키마를 바꿨다 | 마이그레이션 파일 추가 후 `make deploy S=server` (서버가 시작할 때 자동 적용) |
| 환경변수·리소스·replicas 를 바꿨다 | `k8s/` 의 YAML 수정 후 `make up` |
| 비밀값을 바꿨다 | `secrets.env` 수정 → `make up` → `kubectl -n cleansign rollout restart deploy/<이름>` |
| 상태·로그 확인 | `make status`, `kubectl -n cleansign logs deploy/server -f`, 또는 k9s |
| 전부 지우고 다시 | `make down` (DB 데이터까지 삭제) 후 `make up` |

이미지 태그가 항상 `local`이라 YAML 이 바뀌지 않으면 쿠버네티스는 파드를 교체하지 않는다. `make deploy`는 이미지를 넣은 뒤 `rollout restart`까지 해 준다.

## 설계 메모

- **이미지 태그**는 `overlays/local/kustomization.yaml`의 `images:` 한 곳에서만 정한다. 레지스트리를 쓰게 되면 이름과 태그(커밋 SHA)만 바꾼다.
- **비밀값**은 Git에 올리지 않는다(`secrets.env`는 `.gitignore`). Secret은 overlay 의 `secretGenerator`가 만들고, overlay 에도 `namespace: cleansign`이 있어야 한다(없으면 `default`에 생겨서 파드가 "secret not found"로 멈춘다).
- **server**: `RollingUpdate`(surge 1, unavailable 0). 정상 종료 시 진행 중인 분석을 최대 30초 기다리므로 `terminationGracePeriodSeconds: 60`.
- **ai-service**는 `LLM_PROVIDER=mock`이다. 이 VM에서 Ollama 모델은 돌릴 수 없다.
- **web**은 Next.js standalone이고 rewrites가 빌드 시점에 고정된다. `scripts/images.sh`가 `SERVER_URL=http://server:4000`을 빌드 인자로 넣는다. `HOSTNAME=0.0.0.0`을 지정하지 않으면 파드 IP 에만 바인딩되어 `localhost` 접속이 안 된다.

## 실험으로 확인한 것

| 실험 | 결과 |
|---|---|
| postgres 파드 삭제 | 새 파드가 같은 PVC에 붙어서 데이터 유지 |
| 분석 중 server 파드 정상 종료(SIGTERM) | 서버가 진행 중인 분석을 기다려서 완료 |
| 분석 중 server 파드 강제 종료(크래시 가정) | 분석 유실. 15분(`contract.StaleAfter`) 뒤에야 "분석이 중단됐어요"로 표시 |
| server 교체 중 접속 (`Recreate`) | 17개 요청 중 7개 실패(약 3.5초) |
| server 교체 중 접속 (`RollingUpdate`, 현재) | 17개 중 1개 실패 (교체 끝나는 시점. 원인은 확인 못 함) |
| `kubectl rollout undo` | 세 리비전 모두 이미지가 `cleansign/server:local`로 같아서 실제 롤백이 안 됨 |
| ai-service 메모리 | 한도 768Mi에서 OCR 중 `OOMKilled`(exit 137) → 분석 실패. `request 512Mi / limit 2Gi`로 올려서 해결 (실측 157Mi → 완료 직후 374Mi) |

## 알려진 한계

1. **분석이 server 파드 메모리 안에서 돈다.** 크래시나 OOM 으로 파드가 갑자기 죽으면 진행 중이던 분석이 사라진다. 근본 해결은 분석 큐 + worker 분리(Postgres 테이블 큐 + 같은 이미지의 `cmd/worker`)이며, **실사용자 공개 전에 도입**한다. 업로드한 PDF를 어디에 저장할지(DB `bytea`, 볼륨, 오브젝트 스토리지) 먼저 정해야 한다.
2. **롤백이 안 된다.** 태그가 `local`로 고정이라 이전 이미지가 사라진다. 커밋 SHA 태그를 붙이는 CI/CD와 레지스트리를 붙이면 해결된다.
3. **내 맥에서만 배포된다.** 이미지를 VM 에 직접 넣는 방식이라 다른 컴퓨터에서는 쓸 수 없다.
4. **postgres 가 클러스터 안에 있다.** VM 이 사라지면 데이터도 사라진다. 실사용 단계에서는 매니지드 DB 로 옮기고 `DATABASE_URL`만 바꾼다. 백업(`pg_dump` CronJob)도 아직 없다.
5. 분석 점수·등급 정의는 앱 쪽에서 아직 틀려 있다(위험 조항이 있는데 grade 가 낮게 나올 수 있음). 배포와 무관한 앱 이슈다.

## 아직 안 한 것

- `overlays/prod` (새 컴퓨터용): 호스트명, `COOKIE_SECURE=true`, 이미지 출처만 다르게 둘 예정
- GitHub Actions 배포, GHCR(레지스트리), 도메인·DNS·TLS(cert-manager)
- 분석 큐 + worker, 멀티 노드(물리 분리), 모니터링·로그 수집, DB 백업

## 문제가 생겼을 때

| 증상 | 확인 |
|---|---|
| 파드가 `CreateContainerConfigError` | `kubectl -n cleansign describe pod <이름>`의 Events. "secret not found"면 Secret 이 다른 네임스페이스에 생겼는지 확인 |
| 파드가 `OOMKilled`(exit 137)로 재시작 | 메모리 `limits` 초과. `kubectl -n cleansign describe pod`의 Last State 확인 후 한도 조정 |
| `make deploy` 후에도 옛 동작 | 이미지가 VM 에 들어갔는지 `limactl shell k3s-1 sudo k3s ctr images ls -q \| grep cleansign` |
| kubectl 이 `127.0.0.1:<포트> connection refused` | 다른 클러스터(Docker Desktop 등)를 보고 있다. `export KUBECONFIG=~/.lima/k3s-1/copied-from-guest/kubeconfig.yaml` |
| 브라우저가 `https`로 바꿔서 안 열림 | `http://localhost`를 직접 입력 (로컬은 HTTPS 를 설정하지 않았다) |
