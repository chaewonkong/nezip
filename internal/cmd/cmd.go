// Package cmd commands for nezip
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/chaewonkong/nezip/internal/report"
	"github.com/chaewonkong/nezip/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func New(cmds ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nezip",
		Short: "아파트 투자 가치 분석 도구",
	}

	cmd.AddCommand(cmds...)
	return cmd
}

func NewAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "아파트 분석",

		RunE: func(cmd *cobra.Command, args []string) error {
			apt, err := cmd.Flags().GetString("apt")
			if err != nil {
				return err
			}

			if len(apt) == 0 {
				return fmt.Errorf("aptName is required")
			}

			area, err := cmd.Flags().GetFloat64("area")
			if err != nil {
				return err
			}

			if area <= 0.0 {
				return fmt.Errorf("area should greater than 0")
			}

			lawdCD, err := cmd.Flags().GetString("lawd")
			if err != nil {
				return err
			}

			if len(lawdCD) == 0 {
				return fmt.Errorf("lawdCd is required")
			}

			return runAnalyze(cmd.Context(), apt, area, lawdCD)
		},
	}

	cmd.Flags().StringP("apt", "a", "", "아파트명")
	cmd.Flags().Float64("area", 0, "아파트 면적")
	cmd.Flags().StringP("lawd", "l", "", "법정동코드")

	return cmd
}

func runAnalyze(ctx context.Context, aptName string, area float64, lawdCD string) error {
	apiKey := os.Getenv("MOLIT_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("MOLIT_API_KEY 환경변수가 설정되지 않았습니다")
	}

	out, err := service.Analyze(ctx, apiKey, aptName, area, lawdCD)
	if err != nil {
		return err
	}

	return report.WriteJSON(os.Stdout, *out)
}

func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "아파트명 검색",
		RunE: func(cmd *cobra.Command, args []string) error {
			lawdCD, err := cmd.Flags().GetString("lawd")
			if err != nil {
				return err
			}
			if len(lawdCD) == 0 {
				return fmt.Errorf("lawdCd is required")
			}

			return runSearch(cmd.Context(), lawdCD)
		},
	}

	cmd.Flags().StringP("lawd", "l", "", "법정동코드")

	return cmd
}

func runSearch(ctx context.Context, lawdCD string) error {
	if lawdCD == "" {
		return fmt.Errorf("usage: nezip search --lawd <LAWD_CD>")
	}

	apiKey := os.Getenv("MOLIT_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("MOLIT_API_KEY 환경변수가 설정되지 않았습니다")
	}

	names, err := service.Search(ctx, apiKey, lawdCD)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("아파트 목록을 가져올 수 없습니다 (LAWD_CD: %s)", lawdCD)
	}
	for _, name := range names {
		fmt.Println(name)
	}

	return nil
}

func NewMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCP()
		},
	}

	return cmd
}

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

// func runCache(ctx context.Context) {
// 	if len(os.Args) < 3 {
// 		fmt.Fprintln(os.Stderr, "usage: nezip cache <refresh|status>")
// 		os.Exit(1)
// 	}
// 	store, err := service.OpenStore(ctx)
// 	if err != nil {
// 		fmt.Fprintln(os.Stderr, err)
// 		os.Exit(1)
// 	}
// 	defer store.Close()
//
// 	switch os.Args[2] {
// 	case "status":
// 		fmt.Println("cache status: not implemented yet")
// 	case "refresh":
// 		fmt.Println("cache refresh: not implemented yet")
// 	default:
// 		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[2])
// 		os.Exit(1)
// 	}
// }
