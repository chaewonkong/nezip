# nezip

국토교통부 실거래가 API를 이용해 아파트 실거래가 변동율을 조회하고,
강남3구 기준 아파트 대비 상승 추종률을 분석하는 CLI 도구.

Claude Code의 `apt-check` 스킬과 연동하여 사용한다.

## 설치

```bash
go install nezip@latest
```

또는 로컬 빌드:

```bash
go build -o nezip .
```

## 사용법

```bash
nezip <아파트명> <면적㎡> <LAWD_CD>
```

### 예시

```bash
nezip 헬리오시티 84 11710
nezip 마포래미안푸르지오 59 11440
```

## 출력

JSON 형식으로 stdout 출력:

```json
{
  "target": {
    "name": "헬리오시티",
    "area": 84,
    "current": 291500,
    "current_month": "202603",
    "y1": 246500,
    "y1_month": "202503",
    "y1_pct": 18.3,
    "y3": 184500,
    "y3_month": "202303",
    "y3_pct": 58.0
  },
  "gangnam3": {
    "avg_y1_pct": 22.1,
    "avg_y3_pct": 52.4,
    "breakdown": []
  },
  "follow_rate_y1": 82.8,
  "follow_rate_y3": 110.7,
  "cache_used": true,
  "warnings": []
}
```

## 구조

```
nezip/
├── main.go
├── internal/
│   ├── api/      # 국토부 API 호출
│   ├── cache/    # 강남3구 캐시 관리
│   └── calc/     # 변동율·추종율 계산
└── go.mod
```
