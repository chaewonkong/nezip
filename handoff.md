# nezip 작업 핸드오프

## 프로젝트 목적

`~/.claude/skills/apt-check` 스킬이 국토부 API를 WebFetch로 직접 호출하는 구조라
아파트 하나 분석에 10분 가까이 소요됨. 이를 Go CLI 바이너리로 대체해 API 병렬 호출 +
SQLite 캐싱으로 수 초 내 결과를 내는 것이 목표.

---

## 현재 상태

**MCP 서버 전환 완료. v0.2.0 태그 배포.**

- `nezip mcp` 서브커맨드로 MCP stdio 서버 실행
- Claude Code가 `apt_search`, `apt_analyze`를 native tool call로 직접 호출 — E2E 검증 완료
- `internal/service/` 패키지로 핵심 로직 분리 (CLI/MCP 양쪽 재사용)
- MCP 서버 `~/.claude.json`에 등록 완료 (`claude mcp add --scope user`)
- 기존 CLI (`nezip search`, `nezip --apt ...`)는 그대로 유지
- 스킬 이름 `apt-check` → `nezip` 으로 변경
- `SKILL.md` 레포 루트에 추가 (curl로 설치 가능)
- `README.md`에 Claude Code 연동 설치 가이드 추가 (README를 Claude에게 넘기면 자동 설치)

### 설치 (제3자 배포)

README.md의 설치 가이드를 Claude Code에 넘기면 자동으로 실행됨. 수동 순서:

```bash
# 1. 바이너리 설치
go install github.com/chaewonkong/nezip@latest

# 2. MCP 서버 등록
claude mcp add --scope user nezip ~/go/bin/nezip mcp

# 3. 스킬 설치
mkdir -p ~/.claude/skills/nezip
curl -fsSL https://raw.githubusercontent.com/chaewonkong/nezip/main/SKILL.md \
  -o ~/.claude/skills/nezip/SKILL.md

# 4. API 키: ~/.claude/settings.json의 env 섹션에 MOLIT_API_KEY 추가
```

### MCP 서버 검증 (완료)

```
apt_search(lawd_cd: "11710")                                          → 송파구 아파트 목록 정상
apt_analyze(apt_name: "헬리오시티", area: 84, lawd_cd: "11710")       → 분석 JSON 정상
```

---

## 디렉토리 구조

```
nezip/
├── cmd/nezip/
│   ├── main.go          # CLI 진입점 (analyze / search / cache / mcp 서브커맨드)
│   └── mcp.go           # MCP stdio 서버 (apt_search, apt_analyze tool 등록)
├── internal/
│   ├── api/client.go    # 국토부 HTTP 클라이언트 (병렬 goroutine)
│   ├── cache/           # SQLite 캐시 (~/.nezip.db)
│   ├── calc/            # 변동율·추종율 계산
│   ├── db/              # sqlc 생성 코드 (직접 수정 금지)
│   ├── report/          # JSON / --human 출력 포맷
│   └── service/         # 핵심 로직 (Search, Analyze) — CLI·MCP 공유
├── SKILL.md             # nezip 스킬 정의 (curl로 설치)
├── sqlc.yaml
├── go.mod / go.sum
└── README.md
```

---

## CLI 인터페이스

```bash
# 법정동 전체 아파트 목록 조회 (최근 3개월 기준)
nezip search --lawd <LAWD_CD>

# 메인 분석 (JSON stdout)
nezip --apt <아파트명> --area <면적㎡> --lawd <LAWD_CD>

# 직접 사용 시 텍스트 출력
nezip --apt "헬리오시티" --area 84 --lawd 11710 --human

# MCP 서버 실행 (Claude Code가 호출)
nezip mcp

# 캐시 관리 (미구현)
nezip cache refresh
nezip cache status
```

---

## MCP 서버 설정

`~/.claude/mcp.json`은 Claude Code가 읽지 않음. 반드시 `claude mcp add`로 등록해야 함.
설정은 `~/.claude.json`에 저장됨.

```bash
# 등록
claude mcp add --scope user nezip ~/go/bin/nezip mcp

# 등록 확인
claude mcp list
```

`MOLIT_API_KEY`는 `~/.claude/settings.json`의 `env` 섹션에 설정되어 있으므로 별도 전달 불필요.

---

## 환경변수

| 변수 | 설명 |
|------|------|
| `MOLIT_API_KEY` | 국토부 API 인증키 (URL 디코딩된 값) |

`~/.claude/settings.json`의 `env` 섹션에 설정됨.

---

## 데이터 흐름

```
MCP tool call / CLI
  └── service.Analyze() / service.Search()
        ├── fetchAndCacheApt()     # 대상 아파트 36개월 조회·저장
        ├── fetchAndCacheG3()      # 강남3구 기준 아파트 조회·저장
        │     └── 구별 goroutine   # 11650·11680·11710 동시 실행
        ├── calc.AnalyzeWithCurrent()
        ├── calc.Gangnam3Avg()
        ├── calc.FollowRate()
        └── report.Output (JSON)
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

- [ ] `nezip cache refresh` / `cache status` 구현
- [ ] API 키 config 파일 지원 (`~/.config/nezip/config.toml`)
