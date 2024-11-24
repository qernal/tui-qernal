package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	openapi_chaos_client "github.com/qernal/openapi-chaos-go-client"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	qernalToken         = os.Getenv("QERNAL_TOKEN")
	qernalChaosEndpoint = getEnv("QERNAL_HOST_CHAOS", "https://chaos.qernal.com")
	qernalHydraEndpoint = getEnv("QERNAL_HOST_HYDRA", "https://hydra.qernal.com")
	accessToken, _      = _getAccessToken(qernalToken)
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func _getAccessToken(token string) (string, error) {
	if !strings.Contains(token, "@") || strings.Count(token, "@") > 1 {
		err := errors.New("the qernal token is invalid")
		return "", err
	}

	clientId := strings.Split(token, "@")[0]
	clientSecret := strings.Split(token, "@")[1]

	config := clientcredentials.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		TokenURL:     fmt.Sprintf("%s/oauth2/token", qernalHydraEndpoint),
	}

	oauthToken, err := config.Token(context.TODO())
	if err != nil {
		return "", err
	}

	return oauthToken.AccessToken, nil
}

func qernalClient() *openapi_chaos_client.APIClient {
	configuration := &openapi_chaos_client.Configuration{
		Servers: openapi_chaos_client.ServerConfigurations{
			{
				URL: fmt.Sprintf("%s/v1", qernalChaosEndpoint),
			},
		},
		DefaultHeader: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		},
	}

	apiClient := openapi_chaos_client.NewAPIClient(configuration)
	return apiClient
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var logoStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#4C4AD2"))

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	logo := `
 ██████  ███████ ██████  ███    ██  █████  ██
██    ██ ██      ██   ██ ████   ██ ██   ██ ██
██    ██ █████   ██████  ██ ██  ██ ███████ ██
██ ▄▄ ██ ██      ██   ██ ██  ██ ██ ██   ██ ██
 ██████  ███████ ██   ██ ██   ████ ██   ██ ███████
    ▀▀`

	return logoStyle.Render(logo) + "\n" + baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n"
}

func main() {
	// demo project
	// projectRequest := openapi_chaos_client.ApiProjectsFunctionsListRequest{
	// 	projectId: ,
	// }

	client := qernalClient()
	functions, _, _ := client.FunctionsAPI.ProjectsFunctionsList(context.Background(), "{project-id}").Execute()
	fmt.Println(functions)

	columns := []table.Column{
		{Title: "Function", Width: 30},
		{Title: "Type", Width: 4},
		{Title: "Providers", Width: 10},
		{Title: "Regions", Width: 10},
		{Title: "CPU", Width: 4},
		{Title: "Memory", Width: 4},
	}

	rows := []table.Row{}

	for _, function := range functions.Data {
		regionList := []string{}
		providerList := []string{}

		for _, provider := range function.Deployments {
			if !slices.Contains(providerList, *provider.Id) {
				providerList = append(providerList, *provider.Id)
			}

			if !slices.Contains(regionList, *provider.Location.Country) {
				regionList = append(regionList, *provider.Location.Country)
			}
		}

		rows = append(rows, table.Row{
			function.Name,
			string(function.Type),
			strings.Join(providerList, ", "),
			strings.Join(regionList, ", "),
			fmt.Sprint(function.Size.GetCpu()),
			fmt.Sprint(function.Size.GetMemory()),
		})

	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
