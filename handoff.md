# nezip 작업 핸드오프

## 프로젝트 목적

`~/.claude/skills/apt-check` 스킬이 국토부 API를 WebFetch로 직접 호출하는 구조라
아파트 하나 분석에 10분 가까이 소요됨. 이를 Go CLI 바이너리로 대체해 API 병렬 호출 +
SQLite 캐싱으로 수 초 내 결과를 내는 것이 목표.

---

## 현재 상태

**빌드 완료, 바이너리 배포 완료, SKILL.md 업데이트 완료, E2E 검증 완료.**

- `nezip` 바이너리: `~/.claude/skills/apt-check/nezip` (PATH 불필요)
- SKILL.md: search 필수화 + --human 제거 (JSON → Claude 포맷 출력)
- `nezip search`: `--apt` 제거, `--lawd`만으로 전체 목록 반환 (`api.Last3Months()` 사용)
- E2E 검증 완료: `/apt-check 파크타운서안 59` → `파크타운(서안)` 정상 매칭 및 분석

재빌드 후 배포:
```bash
cd /Users/leon/dev/personal/nezip
go build -o nezip ./cmd/nezip
cp nezip ~/.claude/skills/apt-check/nezip
```

동작 검증:
```bash
/apt-check 파크타운서안 59
/apt-check 헬리오시티 84
```

---

## 디렉토리 구조

```
nezip/
├── cmd/nezip/main.go        # CLI 진입점 (analyze / search / cache 서브커맨드)
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
# 법정동 전체 아파트 목록 조회 (최근 3개월 기준)
nezip search --lawd <LAWD_CD>
nezip search --lawd 41135

# 메인 분석 (JSON stdout)
nezip --apt <아파트명> --area <면적㎡> --lawd <LAWD_CD>
nezip --apt "파크타운(서안)" --area 59 --lawd 41135

# 직접 사용 시 텍스트 출력
nezip --apt "헬리오시티" --area 84 --lawd 11710 --human

# 캐시 관리 (미구현)
nezip cache refresh
nezip cache status
```

- `search`: `--apt` 없음. 법정동 전체 목록 반환 → skill(Claude)이 이름 매칭
- `--apt`: API에 등록된 정확한 이름 사용 (괄호 포함, 예: `파크타운(서안)`)
- `--lawd`: 5자리 법정동코드 (SKILL.md 내 테이블 참고)
- `--human`: 텍스트 리포트 출력 (직접 CLI 사용 시). 없으면 JSON stdout

---

## 환경변수

| 변수 | 설명 |
|------|------|
| `MOLIT_API_KEY` | 국토부 API 인증키 (URL 디코딩된 값) |

`~/.claude/settings.json`의 `env` 섹션에 설정됨.

---

## SKILL.md 연동

`~/.claude/skills/apt-check/SKILL.md` 업데이트 완료.

워크플로:
1. 사용자 입력 파싱 (아파트명, 평형 → ㎡)
2. 법정동 코드 추론 (표 조회 또는 WebSearch)
3. `nezip search --lawd <LAWD_CD>` 로 전체 목록 조회 → Claude가 이름 매칭
4. `nezip --apt <정확한 이름> --area <면적> --lawd <LAWD_CD>` 실행 (JSON)
5. JSON 파싱 후 Claude가 보기 좋게 포맷해서 출력

---

## 데이터 흐름

```
main.go
  ├── fetchAndCacheApt()     # 대상 아파트 36개월 조회·저장
  ├── fetchAndCacheG3()      # 강남3구 기준 아파트 조회·저장
  │     └── 구별 goroutine   # 11650·11680·11710 동시 실행
  │           └── FetchMonths() → 병렬 HTTP (동시성 1)
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

면적 기준으로 두 그룹 (각 14~16개 아파트):

- **84㎡ 기준** (54㎡ 미만 또는 64㎡ 초과): 잠실주공5단지, 래미안퍼스티지, 반포자이, 아크로리버파크, 래미안대치팰리스, 헬리오시티 외 10개
- **59㎡ 기준** (54~64㎡): 아크로리버파크, 래미안퍼스티지, 반포자이, 헬리오시티 외 10개

대상 아파트가 강남3구 목록과 겹치면 자동 제외.

---

## 미완료 항목

- [x] `/apt-check` 스킬 E2E 검증 완료 (파크타운서안 59 → 파크타운(서안) 정상 동작)
- [ ] `nezip cache refresh` / `cache status` 구현
- [ ] API 키 config 파일 지원 (`~/.config/nezip/config.toml`)
