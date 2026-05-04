package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/chaewonkong/nezip/internal/report"
	"github.com/chaewonkong/nezip/internal/service"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "mcp":
			if err := runMCP(); err != nil {
				fmt.Fprintln(os.Stderr, "mcp:", err)
				os.Exit(1)
			}
			return
		case "cache":
			runCache(ctx)
			return
		case "search":
			runSearch(ctx)
			return
		}
	}

	var (
		aptName string
		area    float64
		lawdCD  string
		human   bool
	)
	flag.StringVar(&aptName, "apt", "", "아파트명")
	flag.Float64Var(&area, "area", 0, "전용면적(㎡)")
	flag.StringVar(&lawdCD, "lawd", "", "법정동코드")
	flag.BoolVar(&human, "human", false, "사람이 읽는 출력")
	flag.Parse()

	if aptName == "" || area == 0 || lawdCD == "" {
		fmt.Fprintln(os.Stderr, "usage: nezip --apt <아파트명> --area <면적> --lawd <LAWD_CD> [--human]")
		fmt.Fprintln(os.Stderr, "       nezip search --lawd <LAWD_CD>")
		fmt.Fprintln(os.Stderr, "       nezip mcp")
		os.Exit(1)
	}

	if err := runAnalyze(ctx, aptName, area, lawdCD, human); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runAnalyze(ctx context.Context, aptName string, area float64, lawdCD string, human bool) error {
	apiKey := os.Getenv("MOLIT_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("MOLIT_API_KEY 환경변수가 설정되지 않았습니다")
	}

	out, err := service.Analyze(ctx, apiKey, aptName, area, lawdCD)
	if err != nil {
		return err
	}

	if human {
		store, err := service.OpenStore(ctx)
		if err != nil {
			return err
		}
		defer store.Close()
		// WriteHuman은 calc 타입을 직접 받으므로 JSON 출력으로 대체
		return report.WriteJSON(os.Stdout, *out)
	}

	return report.WriteJSON(os.Stdout, *out)
}

func runSearch(ctx context.Context) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	var lawdCD string
	fs.StringVar(&lawdCD, "lawd", "", "법정동코드")
	fs.Parse(os.Args[2:])

	if lawdCD == "" {
		fmt.Fprintln(os.Stderr, "usage: nezip search --lawd <LAWD_CD>")
		os.Exit(1)
	}

	apiKey := os.Getenv("MOLIT_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "MOLIT_API_KEY 환경변수가 설정되지 않았습니다")
		os.Exit(1)
	}

	names, err := service.Search(ctx, apiKey, lawdCD)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(names) == 0 {
		fmt.Fprintf(os.Stderr, "아파트 목록을 가져올 수 없습니다 (LAWD_CD: %s)\n", lawdCD)
		os.Exit(1)
	}
	for _, name := range names {
		fmt.Println(name)
	}
}

func runCache(ctx context.Context) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: nezip cache <refresh|status>")
		os.Exit(1)
	}
	store, err := service.OpenStore(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer store.Close()

	switch os.Args[2] {
	case "status":
		fmt.Println("cache status: not implemented yet")
	case "refresh":
		fmt.Println("cache refresh: not implemented yet")
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[2])
		os.Exit(1)
	}
}
