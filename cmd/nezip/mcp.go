package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/chaewonkong/nezip/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func runMCP() error {
	apiKey := os.Getenv("MOLIT_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("MOLIT_API_KEY 환경변수가 설정되지 않았습니다")
	}

	s := mcpserver.NewMCPServer("nezip", "1.0.0")

	s.AddTool(
		mcp.NewTool("apt_search",
			mcp.WithDescription("법정동 코드로 최근 3개월 아파트 거래 목록을 조회합니다. apt_analyze 호출 전에 정확한 아파트명을 확인하는 데 사용합니다."),
			mcp.WithString("lawd_cd",
				mcp.Required(),
				mcp.Description("5자리 법정동 코드 (예: 41135)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			lawdCD, _ := args["lawd_cd"].(string)
			names, err := service.Search(ctx, apiKey, lawdCD)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(strings.Join(names, "\n")), nil
		},
	)

	s.AddTool(
		mcp.NewTool("apt_analyze",
			mcp.WithDescription("아파트 실거래가 변동율과 강남3구 대비 추종률을 분석합니다. apt_search로 정확한 아파트명을 확인한 뒤 사용하세요."),
			mcp.WithString("apt_name",
				mcp.Required(),
				mcp.Description("API에 등록된 정확한 아파트명 (apt_search로 확인, 예: 파크타운(서안))"),
			),
			mcp.WithNumber("area",
				mcp.Required(),
				mcp.Description("전용면적 ㎡ (예: 84)"),
			),
			mcp.WithString("lawd_cd",
				mcp.Required(),
				mcp.Description("5자리 법정동 코드 (예: 11710)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			aptName, _ := args["apt_name"].(string)
			area, _ := args["area"].(float64)
			lawdCD, _ := args["lawd_cd"].(string)

			out, err := service.Analyze(ctx, apiKey, aptName, area, lawdCD)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			b, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	return mcpserver.ServeStdio(s)
}
