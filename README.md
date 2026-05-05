# nezip

국토교통부 실거래가 API를 이용해 아파트 실거래가 변동율을 조회하고,
강남3구 기준 아파트 대비 상승 추종률을 분석하는 CLI / MCP 서버.

Claude Code의 `nezip` 스킬과 MCP 서버로 연동하여 자연어로 아파트 실거래가를 조회할 수 있다.

## 사전 요구사항

- Claude Code CLI (`claude`)
- 국토부 실거래가 API 키 ([발급](https://www.data.go.kr/data/15058747/openapi.do))

## Claude Code 연동 설치

> Claude Code에 이 README를 넘기면 아래 단계를 대신 실행해줄 수 있다.

### 1. 바이너리 설치

```bash
curl -fsSL https://raw.githubusercontent.com/chaewonkong/nezip/main/install.sh | sh
```

설치 경로를 바꾸고 싶으면:

```bash
INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/chaewonkong/nezip/main/install.sh | sh
```

### 2. MCP 서버 등록

```bash
claude mcp add --scope user nezip ~/go/bin/nezip mcp
```

등록 확인:

```bash
claude mcp list
```

### 3. nezip 스킬 설치

```bash
mkdir -p ~/.claude/skills/nezip
curl -fsSL https://raw.githubusercontent.com/chaewonkong/nezip/main/SKILL.md \
  -o ~/.claude/skills/nezip/SKILL.md
```

### 4. API 키 설정

`~/.claude/settings.json`의 `env` 섹션에 추가:

```json
{
  "env": {
    "MOLIT_API_KEY": "<URL 디코딩된 API 키>"
  }
}
```

> API 키는 공공데이터포털에서 발급 후 URL 디코딩해서 사용. `%2B` → `+` 등.

### 5. Claude Code 재시작

MCP 서버 반영을 위해 Claude Code를 재시작한다.

### 검증

Claude Code에서 직접 확인:

```
apt_search(lawd_cd: "11710")          → 송파구 아파트 목록
apt_analyze(apt_name: "헬리오시티", area: 84, lawd_cd: "11710")  → 분석 결과
```

## nezip 스킬 사용법

설치 후 Claude Code에서 자연어로 요청:

```
/nezip 헬리오시티 84
/nezip 마포래미안푸르지오 59
아파트 실거래가 조회해줘 - 래미안퍼스티지 84㎡
판교 파크타운 59 분석해줘
```

## CLI 직접 사용

MCP 없이 터미널에서 직접 사용할 수도 있다.

### 아파트 목록 조회

```bash
nezip search --lawd <LAWD_CD>
nezip search --lawd 41135
```

### 실거래가 분석

```bash
nezip --apt <아파트명> --area <면적㎡> --lawd <LAWD_CD> [--human]
```

- `--apt`: API에 등록된 정확한 이름 (`search`로 확인 후 사용)
- `--area`: 전용면적(㎡)
- `--lawd`: 5자리 법정동코드
- `--human`: 텍스트 리포트 출력. 없으면 JSON stdout

```bash
nezip search --lawd 41135
nezip --apt "파크타운(서안)" --area 59 --lawd 41135
nezip --apt "헬리오시티" --area 84 --lawd 11710 --human
```

## 출력 (JSON)

```json
{
  "target": {
    "name": "헬리오시티",
    "area": 84,
    "current": 273500,
    "current_month": "202604",
    "y1": 257000,
    "y1_month": "202505",
    "y1_pct": 6.42,
    "y3": 195000,
    "y3_month": "202306",
    "y3_pct": 40.26
  },
  "gangnam3": {
    "avg_y1_pct": 4.15,
    "avg_y3_pct": 39.83,
    "breakdown": [...]
  },
  "follow_rate_y1": 154.6,
  "follow_rate_y3": 101.1,
  "cache_used": false,
  "warnings": []
}
```

## 구조

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
├── SKILL.md             # nezip 스킬 정의
├── sqlc.yaml
└── go.mod
```

## 환경변수

| 변수            | 설명                                         |
| --------------- | -------------------------------------------- |
| `MOLIT_API_KEY` | 국토부 실거래가 API 인증키 (URL 디코딩된 값) |
