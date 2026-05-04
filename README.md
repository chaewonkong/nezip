# nezip

국토교통부 실거래가 API를 이용해 아파트 실거래가 변동율을 조회하고,
강남3구 기준 아파트 대비 상승 추종률을 분석하는 CLI 도구.

Claude Code의 `apt-check` 스킬과 연동하여 사용한다.

## 설치

```bash
go install github.com/chaewonkong/nezip@latest
```

또는 로컬 빌드:

```bash
go build -o nezip ./cmd/nezip
```

## 사용법

### 아파트 목록 조회

법정동 코드로 해당 지역 전체 아파트 목록을 조회한다 (최근 3개월 기준):

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

### 예시

```bash
# 분당구 아파트 목록 확인
nezip search --lawd 41135

# 실거래가 분석 (JSON)
nezip --apt "파크타운(서안)" --area 59 --lawd 41135

# 실거래가 분석 (텍스트)
nezip --apt "헬리오시티" --area 84 --lawd 11710 --human
```

## 출력 (JSON)

```json
{
  "target": {
    "name": "파크타운(서안)",
    "area": 59,
    "current": 135000,
    "current_month": "202604",
    "y1": 105000,
    "y1_month": "202505",
    "y1_pct": 28.57,
    "y3": 87000,
    "y3_month": "202307",
    "y3_pct": 55.17
  },
  "gangnam3": {
    "avg_y1_pct": 9.0,
    "avg_y3_pct": 54.5,
    "breakdown": [...]
  },
  "follow_rate_y1": 317.3,
  "follow_rate_y3": 101.3,
  "cache_used": true,
  "warnings": []
}
```

## 구조

```
nezip/
├── cmd/nezip/main.go        # CLI 진입점 (analyze / search / cache 서브커맨드)
├── internal/
│   ├── api/client.go        # 국토부 HTTP 클라이언트 (병렬 goroutine)
│   ├── cache/               # SQLite 캐시 (~/.nezip.db)
│   ├── calc/                # 변동율·추종율 계산
│   └── report/              # JSON / --human 출력 포맷
├── sqlc.yaml
└── go.mod
```

## 환경변수

| 변수 | 설명 |
|------|------|
| `MOLIT_API_KEY` | 국토부 실거래가 API 인증키 (URL 디코딩된 값) |

## apt-check 스킬 연동

Claude Code의 `/apt-check` 스킬이 이 바이너리를 호출한다:

1. `nezip search --lawd <LAWD_CD>` → 전체 목록에서 Claude가 이름 매칭
2. `nezip --apt <정확한 이름> --area <면적> --lawd <LAWD_CD>` → JSON 분석
3. Claude가 JSON 파싱 후 보기 좋게 포맷해서 출력
