package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type SiliconFlowHandler struct {
	mutex          sync.Mutex
	APIKey         string
	Endpoint       string
	Model          string
	searchApiUrl   string
	searchApiKey   string
	searchApiModel string
	ctx            context.Context
	logger         *logrus.Logger
}

// SiliconFlowRJson shane: Response JSON structure for SiliconFlow
type SiliconFlowRJson struct {
	Choices []struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				Id       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

// SearchOnlineStruct shane: Big Model for search online
type SearchOnlineStruct struct {
	Choices []struct {
		Message struct {
			ToolCalls []struct {
				Id           string         `json:"id"`
				SearchResult []SearchResult `json:"search_result"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

type SFMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallId string `json:"tool_call_id,omitempty"`
}

// SFTools shane: function_call structure for SiliconFlow
type SFTools []struct {
	Function struct {
		Description string      `json:"description"`
		Name        string      `json:"name"`
		Parameters  interface{} `json:"parameters"`
		Required    []string    `json:"required"`
	} `json:"function"`
	Type string `json:"type"`
}

type SFRequest struct {
	Model       string      `json:"model"`
	Messages    []SFMessage `json:"messages"`
	MaxTokens   int         `json:"max_tokens"`
	Stream      bool        `json:"stream,omitempty"`
	Temperature float64     `json:"temperature,omitempty"`
	Tools       SFTools     `json:"tools,omitempty"`
}

type SFChoice struct {
	Message struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		ToolCalls []struct {
			Id       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls,omitempty"`
	} `json:"message"`
}

type SFResponse struct {
	Choices []SFChoice `json:"choices"`
}

// QueryParameters shane: Query parameters
type QueryParameters struct {
	Query struct {
		Description string `json:"description"`
		Type        string `json:"type"`
	} `json:"query"`
}

// PromptParameters shane: Prompt parameters
type PromptParameters struct {
	Prompt struct {
		Description string `json:"description"`
		Type        string `json:"type"`
	} `json:"prompt"`
}

// RespData shane: Response
type RespData struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type SearchResult struct {
	Content string `json:"content"`
	Icon    string `json:"icon"`
	Index   int    `json:"index"`
	Link    string `json:"link"`
	Media   string `json:"media"`
	Refer   string `json:"refer"`
	Title   string `json:"title"`
}

type WeatherResponse struct {
	Precipitation       float64 `json:"precipitation"`
	Temperature         float64 `json:"temperature"`
	Pressure            int     `json:"pressure"`
	Humidity            int     `json:"humidity"`
	WindDirection       string  `json:"windDirection"`
	WindDirectionDegree int     `json:"windDirectionDegree"`
	WindSpeed           float64 `json:"windSpeed"`
	WindScale           string  `json:"windScale"`
	Feelst              float64 `json:"feelst"`
	Code                int     `json:"code"`
	Place               string  `json:"place"`
	Weather1            string  `json:"weather1"`
	Weather2            string  `json:"weather2"`
	Weather1img         string  `json:"weather1img"`
	Weather2img         string  `json:"weather2img"`
	Uptime              string  `json:"uptime"`
	Jieqi               string  `json:"jieqi"`
}

func NewSiliconFlowHandler(ctx context.Context, apiKey, endpoint, model string, logger *logrus.Logger, searchApiUrl string, searchApiKey string, searchApiModel string) *SiliconFlowHandler {
	return &SiliconFlowHandler{
		ctx:            ctx,
		APIKey:         apiKey,
		Endpoint:       endpoint,
		Model:          model,
		logger:         logger,
		searchApiUrl:   searchApiUrl,
		searchApiKey:   searchApiKey,
		searchApiModel: searchApiModel,
	}
}

func (h *SiliconFlowHandler) Query(userMsg string) (string, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// shane: define the tools
	tools := SFTools{
		{
			Type: "function",
			Function: struct {
				Description string      `json:"description"`
				Name        string      `json:"name"`
				Parameters  interface{} `json:"parameters"`
				Required    []string    `json:"required"`
			}{
				Description: "The function sends a query to the browser and returns relevant results based on the search terms provided. The model should avoid using this function if it already possesses the required information or can provide a confident answer without external data",
				Name:        "searchOnline",
				Parameters: QueryParameters{
					Query: struct {
						Description string `json:"description"`
						Type        string `json:"type"`
					}{
						Description: "What to search for",
						Type:        "string",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Type: "function",
			Function: struct {
				Description string      `json:"description"`
				Name        string      `json:"name"`
				Parameters  interface{} `json:"parameters"`
				Required    []string    `json:"required"`
			}{
				Description: "Generate an image based on a given prompt",
				Name:        "generateImage",
				Parameters: PromptParameters{
					Prompt: struct {
						Description string `json:"description"`
						Type        string `json:"type"`
					}{
						Description: "A text prompt describing the image to be generated",
						Type:        "string",
					},
				},
				Required: []string{"prompt"},
			},
		},
		{
			Type: "function",
			Function: struct {
				Description string      `json:"description"`
				Name        string      `json:"name"`
				Parameters  interface{} `json:"parameters"`
				Required    []string    `json:"required"`
			}{
				Description: "查询指定地点的天气信息,需要剥离出省份信息放到sheng参数里面,并且需要剥离出地点的信息放到place参数里面",
				Name:        "queryWeather",
				Parameters: struct {
					Sheng string `json:"sheng"`
					Place string `json:"place"`
				}{
					Sheng: "",
					Place: "",
				},
				Required: []string{"sheng", "place"},
			},
		},
	}

	reqBody := SFRequest{
		Model:     h.Model,
		Messages:  []SFMessage{{Role: "user", Content: userMsg}},
		MaxTokens: 512,
		Stream:    false,
		Tools:     tools,
	}
	body, _ := json.Marshal(reqBody)

	// shane: siliconflow request
	req, _ := http.NewRequestWithContext(h.ctx, "POST", h.Endpoint, bytes.NewReader(body))
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", "Bearer "+h.APIKey)
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	logrus.Infof("LLM respBody:%s", string(respBody))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var sfResp SFResponse
	if err := json.Unmarshal(respBody, &sfResp); err != nil {
		return "", err
	}
	if len(sfResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// shane: Check if the response contains tool calls
	if len(sfResp.Choices[0].Message.ToolCalls) > 0 {
		toolCall := sfResp.Choices[0].Message.ToolCalls[0]
		funcName := toolCall.Function.Name
		arguments := toolCall.Function.Arguments
		toolCallId := toolCall.Id

		// shane: handle the tool call
		switch funcName {
		case "searchOnline":
			logrus.Info("[Handling searchOnline tool call]")
			return h.handleSearchOnline(arguments, userMsg, toolCallId)
		case "generateImage":
			// shane: handle image generation
			logrus.Info("[Handling generateImage tool call]")
			return h.handleGenerateImage(arguments)
		case "queryWeather":
			logrus.Info("[Handling queryWeather tool call]")
			return h.handleQueryWeather(arguments, userMsg, toolCallId)
		default:
			logrus.Info("[Handling no tool call]")
			return "", fmt.Errorf("unknown function: %s", funcName)
		}
	}

	// shane: If no tool calls, return the raw content
	return sfResp.Choices[0].Message.Content, nil
}

func (h *SiliconFlowHandler) handleSearchOnline(arguments string, userMsg string, toolCallId string) (string, error) {
	argumentsJson := struct {
		Query string `json:"query"`
	}{}
	if err := json.Unmarshal([]byte(arguments), &argumentsJson); err != nil {
		h.logger.Errorf("Failed to unmarshal search online arguments: %v", err)
		return "", err
	}
	searchOnline, err := h.SearchOnline(argumentsJson.Query)
	if err != nil {
		h.logger.Errorf("Failed to search online arguments: %v", err)
		return "", err
	}
	if len(searchOnline.Choices) == 0 || len(searchOnline.Choices[0].Message.ToolCalls) == 0 {
		return "", fmt.Errorf("no search results found")
	}
	searchResult := searchOnline.Choices[0].Message.ToolCalls[0].SearchResult
	// shane:
	reqBody := SFRequest{
		Model: h.Model,
		Messages: []SFMessage{
			{Role: "system", Content: "Please provide the user's question and the specific search results, and I will directly use the search results to answer the question in English, summarizing naturally if there are multiple sources."},
			{Role: "user", Content: userMsg},
			{Role: "tool", Content: h.searchResultToString(searchResult), ToolCallId: toolCallId},
		},
		MaxTokens: 512,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(h.ctx, "POST", h.Endpoint, bytes.NewReader(body))
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", "Bearer "+h.APIKey)
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var sfResp SFResponse
	if err := json.Unmarshal(respBody, &sfResp); err != nil {
		return "", err
	}
	if len(sfResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return sfResp.Choices[0].Message.Content, nil
}

func (h *SiliconFlowHandler) handleGenerateImage(arguments string) (string, error) {
	argumentsJson := struct {
		Prompt string `json:"prompt"`
	}{}
	if err := json.Unmarshal([]byte(arguments), &argumentsJson); err != nil {
		h.logger.Errorf("Failed to unmarshal image generation arguments: %v, arguments = %s", err, string(arguments))
		return "", err
	}

	// shane: Construct the image URL
	imageUrl := fmt.Sprintf("https://image.pollinations.ai/prompt/%s?width=1024&height=1024&seed=100&model=flux&nologo=true", argumentsJson.Prompt)

	// shane: md style
	return fmt.Sprintf("![%s](%s)", argumentsJson.Prompt, imageUrl), nil
}

func (h *SiliconFlowHandler) SearchOnline(query string) (SearchOnlineStruct, error) {
	searchJson := map[string]interface{}{
		"assistant_id": "659e54b1b8006379b4b2abd6",
		"model":        "glm-4v-flash",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": query,
					},
				},
			},
		},
		"stream":      true,
		"temperature": 0.2,
	}
	marshal, err := json.Marshal(searchJson)
	if err != nil {
		fmt.Println(err)
		logrus.Error(err)
		return SearchOnlineStruct{}, err
	}
	request, err := http.NewRequest("POST", h.searchApiUrl, bytes.NewReader(marshal))
	if err != nil {
		return SearchOnlineStruct{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+h.searchApiKey)
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return SearchOnlineStruct{}, err
	}
	defer response.Body.Close()

	reader := bufio.NewReader(response.Body)

	// shane: WebBrowserOutput regex
	webBrowserOutputRegex := regexp.MustCompile(`"web_browser":\s*\{"outputs":\s*(\[.*?\])\}`)

	var combinedData string
	var searchResults []SearchResult
	for {
		line, err := reader.ReadBytes('\n') // shane: read line by line
		if err != nil {
			if err == io.EOF {
				break
			}
			return SearchOnlineStruct{}, err
		}
		lineStr := strings.TrimSpace(string(line))
		if lineStr == "" || !strings.HasPrefix(lineStr, "data: ") {
			continue
		}
		data := strings.TrimPrefix(lineStr, "data: ")
		if data == "[DONE]" {
			break
		}
		combinedData += data
	}

	webBrowserMatches := webBrowserOutputRegex.FindStringSubmatch(combinedData)
	if len(webBrowserMatches) > 1 {
		outputsStr := webBrowserMatches[1]
		cleanStr := strings.ReplaceAll(outputsStr, "\\\"", "\"") // shane: remove \"

		// shane: parse the result
		var outputs []map[string]interface{}
		err = json.Unmarshal([]byte(cleanStr), &outputs)
		if err != nil {
			// shane: if parsing fails, return an error with the original content
			return SearchOnlineStruct{
				Choices: []struct {
					Message struct {
						ToolCalls []struct {
							Id           string         `json:"id"`
							SearchResult []SearchResult `json:"search_result"`
						} `json:"tool_calls"`
					} `json:"message"`
				}{
					{
						Message: struct {
							ToolCalls []struct {
								Id           string         `json:"id"`
								SearchResult []SearchResult `json:"search_result"`
							} `json:"tool_calls"`
						}{
							ToolCalls: []struct {
								Id           string         `json:"id"`
								SearchResult []SearchResult `json:"search_result"`
							}{
								{
									Id: "search_result",
									SearchResult: []SearchResult{{
										Title:   "搜索结果原始内容",
										Content: fmt.Sprintf("搜索结果原始内容: %s", cleanStr),
										Link:    "",
									}},
								},
							},
						},
					},
				},
			}, nil
		}

		// shane: convert to SearchResult
		for _, item := range outputs {
			result := SearchResult{}
			if title, ok := item["title"].(string); ok {
				result.Title = title
			}
			if content, ok := item["content"].(string); ok {
				result.Content = content
			}
			if link, ok := item["link"].(string); ok {
				result.Link = strings.TrimSpace(link)
			}
			if index, ok := item["index"].(float64); ok {
				result.Index = int(index)
			}
			if icon, ok := item["icon"].(string); ok {
				result.Icon = icon
			}
			if media, ok := item["media"].(string); ok {
				result.Media = media
			}
			if refer, ok := item["refer"].(string); ok {
				result.Refer = refer
			}

			searchResults = append(searchResults, result)
		}
	}

	result := SearchOnlineStruct{
		Choices: []struct {
			Message struct {
				ToolCalls []struct {
					Id           string         `json:"id"`
					SearchResult []SearchResult `json:"search_result"`
				} `json:"tool_calls"`
			} `json:"message"`
		}{
			{
				Message: struct {
					ToolCalls []struct {
						Id           string         `json:"id"`
						SearchResult []SearchResult `json:"search_result"`
					} `json:"tool_calls"`
				}{
					ToolCalls: []struct {
						Id           string         `json:"id"`
						SearchResult []SearchResult `json:"search_result"`
					}{
						{
							Id:           "search_result",
							SearchResult: searchResults,
						},
					},
				},
			},
		},
	}

	return result, nil
}

func (h *SiliconFlowHandler) handleQueryWeather(arguments string, userMsg string, toolCallId string) (string, error) {
	args := struct {
		Sheng string `json:"sheng"`
		Place string `json:"place"`
	}{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		h.logger.Errorf("Failed to unmarshal weather query arguments: %v", err)
		return "", err
	}
	// shane: construct the request data
	data := fmt.Sprintf("id=10006512&key=512b69d6b44c1c59a1a698da8d3cb1a7&sheng=%s&place=%s", args.Sheng, args.Place)
	request, err := http.NewRequest("POST", "https://cn.apihz.cn/api/tianqi/tqyb.php", bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var weatherResp WeatherResponse
	if err := json.Unmarshal(respBody, &weatherResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal weather response: %v, content: %s", err, string(respBody))
	}
	// shane: construct the weather information string
	weatherInfo := fmt.Sprintf("当前时间：%s,%s 的天气情况如下：温度为 %.1f 摄氏度，天气状况为 %s，风力为 %s，相对湿度为 %d%%。",
		weatherResp.Uptime, weatherResp.Place, weatherResp.Temperature, weatherResp.Weather1, weatherResp.WindScale, weatherResp.Humidity)

	reqBody := SFRequest{
		Model: h.Model,
		Messages: []SFMessage{
			{Role: "system", Content: "Please use the following weather information to answer the user's question. Ensure your response is concise and clear. The weather details include precipitation, temperature, pressure, humidity, wind direction, wind speed, wind scale, feels-like temperature, location, weather conditions (such as light rain or moderate rain), update time, etc. You must reply in Chinese."},
			{Role: "user", Content: userMsg},
			{Role: "tool", Content: weatherInfo, ToolCallId: toolCallId},
		},
		MaxTokens: 512,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(h.ctx, "POST", h.Endpoint, bytes.NewReader(body))
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", "Bearer "+h.APIKey)
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var sfResp SFResponse
	if err := json.Unmarshal(respBody, &sfResp); err != nil {
		return "", err
	}
	if len(sfResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return sfResp.Choices[0].Message.Content, nil
}

// shane: struct2string
func (h *SiliconFlowHandler) searchResultToString(results []SearchResult) string {
	var buf bytes.Buffer
	for _, r := range results {
		buf.WriteString(fmt.Sprintf("Title: %s\nContent: %s\nLink: %s\n\n", r.Title, r.Content, r.Link))
	}
	return buf.String()
}
