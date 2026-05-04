# nezip 작업 핸드오프

## 프로젝트 목적

`~/.claude/skills/apt-check` 스킬이 국토부 API를 WebFetch로 직접 호출하는 구조라
아파트 하나 분석에 10분 가까이 소요됨. 이를 Go CLI 바이너리로 대체해 API 병렬 호출 +
SQLite 캐싱으로 수 초 내 결과를 내는 것이 목표.

---

## 현재 상태

**빌드 완료, 테스트 통과. 실제 API 호출 테스트 미완료.**

Claude Code Bash tool의 sandbox가 네트워크를 차단해 직접 실행 검증이 안 됨.
`~/.claude/settings.json`에 `sandbox.network.allowedDomains: ["apis.data.go.kr"]` 추가했으니
**Claude Code 재시작 후** 테스트 필요.

```bash
cd /Users/leon/dev/personal/nezip
go run ./cmd --apt 파크타운서안 --area 59 --lawd 41135 --human
```

---

## 디렉토리 구조

```
nezip/
├── cmd/main.go              # CLI 진입점 + 전체 오케스트레이션
├── internal/
│   ├── api/client.go        # 국토부 HTTP 클라이언트 (병렬 goroutine)
│   ├── cache/
│   │   ├── store.go         # SQLite Store 래퍼
│   │   ├── schema.sql       # prices 테이블 DDL
│   │   └── query.sql        # sqlc 쿼리 정의
│   ├── calc/
│   │   ├── calc.go          # 변동율·추종율 계산 로직
│   │   └── calc_test.go     # 단위 테스트
│   ├── db/                  # sqlc 생성 코드 (직접 수정 금지)
│   │   ├── db.go
│   │   ├── models.go
│   │   └── query.sql.go
│   └── report/report.go     # JSON / --human 출력 포맷
├── sqlc.yaml
├── go.mod / go.sum
└── README.md
```

---

## CLI 인터페이스

```bash
# 메인 분석
nezip --apt <아파트명> --area <면적㎡> --lawd <LAWD_CD>
nezip --apt 헬리오시티 --area 84 --lawd 11710 --human

# 캐시 관리 (미구현)
nezip cache refresh
nezip cache status
```

- `--apt`: `strings.Contains` 매칭. 긴 이름이면 핵심 키워드만 넣어도 됨
- `--lawd`: 5자리 법정동코드 (SKILL.md 내 테이블 참고)
- `--human`: 텍스트 리포트 출력. 없으면 JSON stdout

---

## 환경변수

| 변수 | 설명 |
|------|------|
| `MOLIT_API_KEY` | 국토부 API 인증키 (URL 디코딩된 값) |

`~/.claude/settings.json`의 `env` 섹션에 이미 설정됨.

---

## 데이터 흐름

```
main.go
  ├── fetchAndCacheApt()     # 대상 아파트 36개월 조회·저장
  ├── fetchAndCacheG3()      # 강남3구 기준 아파트 조회·저장
  │     └── 구별 goroutine   # 11650·11680·11710 동시 실행
  │           └── FetchMonths() → 36개월 병렬 HTTP
  ├── calc.AnalyzeWithCurrent()
  ├── calc.Gangnam3Avg()
  ├── calc.FollowRate()
  └── report.WriteJSON / WriteHuman
```

---

## DB 스키마

파일 위치: `~/.nezip.db`

```sql
CREATE TABLE prices (
    apt_name TEXT    NOT NULL,
    lawd_cd  TEXT    NOT NULL,
    month    TEXT    NOT NULL,  -- YYYYMM
    area_int INTEGER NOT NULL,  -- ROUND(excluUseAr)
    area     REAL    NOT NULL,
    median   INTEGER NOT NULL,
    count    INTEGER NOT NULL,
    PRIMARY KEY (apt_name, lawd_cd, area_int, month)
);
```

**캐시 전략**:
- 당월: `INSERT OR REPLACE` (upsert)
- 과거월: `INSERT OR IGNORE` (중복 스킵)
- 만료 없음 — 과거 실거래가는 소급 변경되지 않으므로 영구 보관

---

## 강남3구 기준 아파트

면적 기준으로 두 그룹:

- **84㎡ 기준** (54㎡ 미만 또는 64㎡ 초과): 16개 아파트 (잠실주공5단지, 래미안퍼스티지, 반포자이, 아크로리버파크, 래미안대치팰리스, 헬리오시티 + 10개)
- **59㎡ 기준** (54~64㎡): 14개 아파트

대상 아파트가 강남3구 목록과 겹치면 자동 제외.

---

## 미완료 항목

- [ ] `nezip cache refresh` / `cache status` 구현
- [ ] API 키 config 파일 지원 (`~/.config/nezip/config.toml`)
- [ ] Claude Code 재시작 후 실제 API 호출 검증
- [ ] SKILL.md 업데이트 (WebFetch → `nezip` 바이너리 호출로 변경)
- [ ] `go install` 등록 및 `$PATH` 확인

---

## SKILL.md 연동 계획

현재 SKILL.md는 Claude가 WebFetch로 직접 API를 호출함.
바이너리 검증 완료 후 SKILL.md의 STEP 2~5를 아래로 교체:

```
STEP 2~5 → Bash tool로 nezip 실행:
  nezip --apt {아파트명} --area {면적} --lawd {LAWD_CD}
  결과 JSON을 받아 STEP 7 리포트 포맷팅만 수행
```
