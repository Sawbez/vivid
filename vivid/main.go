package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

var (
	term = termenv.EnvColorProfile()
)

type colorMsg struct{ result [5][3]int }
type errMsg struct{ err error }
type modelMsg struct{ result []string }

func (e errMsg) Error() string { return e.err.Error() }

type color_response struct{ Result [5][3]int }

type model_response struct{ Result []string }

type model struct {
	tabContent [5][3]int
	lockedTabs [5]bool
	activeTab  int
	err        error
	width      int
	height     int
	quitting   bool
	editing    bool
	color_text [3]string
	edit_tab   int
	models     []string
	model_loc  int
	done_model bool
	model_type string
}

func (m model) Init() tea.Cmd {
	return getModels
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.done_model {
		switch msg := msg.(type) {
		case modelMsg:
			m.models = msg.result

		case tea.WindowSizeMsg:
			m.width, m.height = msg.Width, msg.Height

		case tea.KeyMsg:
			keypress := msg.String()
			switch keypress {
			case "ctrl+c", "esc", "q":
				m.quitting = true
				return m, tea.Quit
			case "up", "w":
				m.model_loc = wrap_move(m.model_loc, -1, len(m.models))
				return m, nil
			case "down", "s":
				m.model_loc = wrap_move(m.model_loc, 1, len(m.models))
				return m, nil
			case "enter":
				if len(m.models) > 0 {
					m.model_type = m.models[m.model_loc]
					m.done_model = true
				}
				return m, nil
			}

		case errMsg:
			m.err = msg.err
			return m, tea.Quit
		}
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		keypress := msg.String()
		switch keypress {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit
		case "e":
			if m.done_model {
				if !m.editing {
					var temp [3]string
					for i := 0; i < 3; i++ {
						temp[i] = strconv.Itoa(m.tabContent[m.activeTab][i])
					}
					m.color_text = temp
				}
				m.editing = !m.editing
			}
		case "m":
			if !m.editing {
				m.done_model = false
			}
		}

		if !m.editing {
			switch keypress {
			case "enter", " ":
				m.lockedTabs[m.activeTab] = !m.lockedTabs[m.activeTab]
				return m, nil
			case "right", "d":
				m.activeTab = wrap_move(m.activeTab, 1, len(m.tabContent))
				return m, nil
			case "left", "a":
				m.activeTab = wrap_move(m.activeTab, -1, len(m.tabContent))
				return m, nil
			case "<", ",":
				newIndex := wrap_move(m.activeTab, -1, len(m.tabContent))
				m.tabContent[m.activeTab], m.tabContent[newIndex] = m.tabContent[newIndex], m.tabContent[m.activeTab]
				return m, nil
			case ">", ".":
				newIndex := wrap_move(m.activeTab, 1, len(m.tabContent))
				m.tabContent[m.activeTab], m.tabContent[newIndex] = m.tabContent[newIndex], m.tabContent[m.activeTab]
				return m, nil
			case "r":
				return m, getColors(m.model_type, m.tabContent, m.lockedTabs)
			}
		} else {
			switch keypress {
			case "enter":
				color, err := get_color(m.color_text)
				if err != nil {
					return m, nil
				}
				m.tabContent[m.activeTab] = color
				m.editing = false
				m.color_text = [3]string{"", "", ""}
				return m, nil
			case "up", "w":
				m.edit_tab = wrap_move(m.edit_tab, -1, 3)
				return m, nil
			case "down", "s":
				m.edit_tab = wrap_move(m.edit_tab, 1, 3)
				return m, nil
			case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
				m.color_text[m.edit_tab] += keypress
				return m, nil
			case "backspace":
				sz := len(m.color_text[m.edit_tab])
				if sz > 0 {
					m.color_text[m.edit_tab] = m.color_text[m.edit_tab][:sz-1]
				}
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case errMsg:
		m.err = msg.err
		return m, tea.Quit

	case colorMsg:
		m.tabContent = msg.result
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("An error occured:\n%s\n", m.err)
	}
	if m.quitting {
		small_colorbar := ""
		readable_rgb := ""
		prev_l := 0
		for i := 0; i < len(m.tabContent); i++ {
			temp_chr := makeColorChar(m.tabContent[i], [3]int{0, 0, 0}, " ")
			readable_rgb += fmt.Sprintf(`[%d,%d,%d]`, m.tabContent[i][0], m.tabContent[i][1], m.tabContent[i][2])
			if i != len(m.tabContent)-1 {
				readable_rgb += ","
			}
			str_len := len(readable_rgb)
			small_colorbar += strings.Repeat(temp_chr, str_len-prev_l)
			prev_l = str_len
		}
		return small_colorbar + "\n" + readable_rgb + "\n"
	}
	if !m.done_model {
		if len(m.models) == 0 {
			return "Loading models..."
		} else {
			result := ""
			for i := 0; i < len(m.models); i++ {
				temp := " > "
				temp += m.models[i]
				if i == m.model_loc {
					temp = termenv.Style{}.Foreground(term.Color("#56DE56")).Styled(temp)
				}
				result += temp + "\n"
			}
			return result + "\n"
		}
	}
	if !m.editing {
		colorbar := ""
		barWidth := max(m.width, len(m.tabContent)) / len(m.tabContent)
		missingSquares := max(m.width, len(m.tabContent)) % len(m.tabContent)
		barHeight := max(m.height, 1)
		var tabWidth int
		for j := 0; j < barHeight; j++ {
			for i := 0; i < len(m.tabContent); i++ {
				tabWidth = barWidth
				if i == len(m.tabContent)/2 {
					tabWidth += missingSquares
				}
				// ┌─┐
				// │ │
				// └─┘
				if i == m.activeTab || m.lockedTabs[i] {
					var lookup [3][3]string
					if i == m.activeTab && m.lockedTabs[i] {
						lookup = [3][3]string{
							{"╔", "═", "╗"},
							{"║", " ", "║"},
							{"╚", "═", "╝"},
						}
					} else if m.lockedTabs[i] {
						lookup = [3][3]string{
							{"┌", "─", "┐"},
							{"│", " ", "│"},
							{"└", "─", "┘"},
						}
					} else {
						lookup = [3][3]string{
							{"┏", "━", "┓"},
							{"┃", " ", "┃"},
							{"┗", "━", "┛"},
						}
					}
					ringColor := [3]int{255 - m.tabContent[i][0], 255 - m.tabContent[i][1], 255 - m.tabContent[i][2]}
					if barHeight == 1 {
						colorbar += makeColorChar(m.tabContent[i], ringColor, "│")
					} else {
						var nextChrs [3]string
						if j == 1 {
							nextChrs = lookup[0]
						} else if j != barHeight-1 {
							nextChrs = lookup[1]
						} else {
							nextChrs = lookup[2]
						}
						colorbar += makeColorChar(m.tabContent[i], ringColor, nextChrs[0])
						colorbar += strings.Repeat(makeColorChar(m.tabContent[i], ringColor, nextChrs[1]), tabWidth-2)
						colorbar += makeColorChar(m.tabContent[i], ringColor, nextChrs[2])
					}
				} else {
					colorbar += strings.Repeat(makeColorChar(m.tabContent[i], [3]int{0, 0, 0}, " "), tabWidth)
				}

			}
			colorbar += "\n"
		}

		return colorbar
	} else {
		out := ""
		color, err := get_color(m.color_text)
		valid_color := err == nil
		for i := 0; i < 3; i++ {
			temp := " > "
			temp += m.color_text[i]
			if i == m.edit_tab {
				temp = termenv.Style{}.Foreground(term.Color("#56DE56")).Styled(temp)
				temp += " "
				if valid_color {
					temp += termenv.Style{}.Background(term.Color(fmt.Sprintf("#%02x%02x%02x", color[0], color[1], color[2]))).Styled(" ")
				} else {
					temp += "x"
				}
			}
			out += temp
			out += "\n"
		}
		return out + "\n"
	}
}

func main() {
	lockedTabsDefault := [5]bool{false, false, false, false, false}
	var tabContentStart [5][3]int
	m := model{lockedTabs: lockedTabsDefault, tabContent: tabContentStart, quitting: false}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func wrap_move(current, change, max int) int {
	if current+change > max-1 {
		return 0
	} else if current+change < 0 {
		return max - 1
	} else {
		return current + change
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getColors(model string, prev [5][3]int, locks [5]bool) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 5 * time.Second}

		lock_data := ""
		for i := 0; i < len(prev); i++ {
			if locks[i] {
				lock_data += fmt.Sprintf(`[%d,%d,%d]`, prev[i][0], prev[i][1], prev[i][2])
			} else {
				lock_data += `"N"`
			}
			if i != len(prev)-1 {
				lock_data += ","
			}
		}
		data := strings.NewReader(`{"model":"` + model + `","input":[` + lock_data + `]}`)
		req, req_err := http.NewRequest("POST", "http://colormind.io/api/", data)
		if req_err != nil {
			return errMsg{req_err}
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, resp_err := client.Do(req)
		if resp_err != nil {
			return errMsg{resp_err}
		}
		defer resp.Body.Close()
		color_responseText, read_err := io.ReadAll(resp.Body)
		if read_err != nil {
			return errMsg{read_err}
		}
		color_responseData := color_response{}
		json_err := json.Unmarshal(color_responseText, &color_responseData)
		if json_err != nil {
			return errMsg{json_err}
		}
		return colorMsg{color_responseData.Result}
	}
}

func getModels() tea.Msg {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, resp_err := client.Get("http://colormind.io/list/")
	if resp_err != nil {
		return errMsg{resp_err}
	}
	defer resp.Body.Close()
	model_responseText, read_err := io.ReadAll(resp.Body)
	if read_err != nil {
		return errMsg{read_err}
	}
	model_responseData := model_response{}
	json_err := json.Unmarshal(model_responseText, &model_responseData)
	if json_err != nil {
		return errMsg{json_err}
	}
	return modelMsg{model_responseData.Result}
}

func makeColorChar(c, c2 [3]int, ch string) string {
	return termenv.Style{}.Background(
		term.Color(fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2]))).Foreground(
		term.Color(fmt.Sprintf("#%02x%02x%02x", c2[0], c2[1], c2[2]))).Styled(ch)
}

func get_color(col [3]string) ([3]int, error) {
	var out [3]int
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(col[i])
		if err != nil || n > 255 || n < 0 {
			return [3]int{0, 0, 0}, &strconv.NumError{}
		}
		out[i] = n
	}
	return out, nil
}
