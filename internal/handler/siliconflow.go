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
	"time"

	"github.com/google/uuid"
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
	history        []SFMessage
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
			Content   string `json:"content"`
			ToolCalls []struct {
				Id       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
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

// TextTransformRequest
type TextTransformRequest struct {
	Model string `json:"model"`
	Input struct {
		Text   string `json:"text"`
		Prompt string `json:"prompt"`
	} `json:"input"`
	Parameters struct {
		Steps            int    `json:"steps"`
		N                int    `json:"n"`
		FontName         string `json:"font_name,omitempty"`
		TtfUrl           string `json:"ttf_url,omitempty"`
		OutputImageRatio string `json:"output_image_ratio"`
	} `json:"parameters"`
}

// TextTransformResponse
type TextTransformResponse struct {
	Output struct {
		TaskId     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
	} `json:"output"`
	Usage struct {
		ImageCount int `json:"image_count"`
	} `json:"usage"`
	RequestId string `json:"request_id"`
}

// TextTransformTaskQuery
type TaskQueryResponse struct {
	Output struct {
		TaskId     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		Results    []struct {
			SvgUrl string `json:"svg_url"`
			PngUrl string `json:"png_url"`
		} `json:"results"`
	} `json:"output"`
	Usage struct {
		ImageCount int `json:"image_count"`
	} `json:"usage"`
	RequestId string `json:"request_id"`
}

// Ine: TextTransform parameters
type TextTransformParameters struct {
	Text struct {
		Description string `json:"description"`
		Type        string `json:"type"`
	} `json:"text"`
	Prompt struct {
		Description string `json:"description"`
		Type        string `json:"type"`
	}
	Style struct {
		Description string `json:"description"`
		Type        string `json:"type"`
	} `json:"style"`
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

func (h *SiliconFlowHandler) QueryStream(userMsg string, ttsCallback func(segment string, playID string, autoHangup bool) error) (string, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.history = append(h.history, SFMessage{
		Role:    "user",
		Content: userMsg,
	})

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
				Description: "查询指定地点的天气信息,地点必须是中文，不能是英文！需要剥离出省份信息放到sheng参数里面,并且需要剥离出地点的信息放到place参数里面",
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
				Description: "对输入的文字进行艺术变形处理，支持多种变形样式如花体字、艺术字等，可以用于装饰性文字展示",
				Name:        "transformText",
				Parameters: TextTransformParameters{
					Text: struct {
						Description string `json:"description"`
						Type        string `json:"type"`
					}{
						Description: "需要进行变形处理的文字内容",
						Type:        "string",
					},
					Prompt: struct {
						Description string `json:"description"`
						Type        string `json:"type"`
					}{
						Description: "艺术字风格描述提示词",
						Type:        "string",
					},
					Style: struct {
						Description string `json:"description"`
						Type        string `json:"type"`
					}{
						Description: "变形样式，如flower(花体)、art(艺术字)、gothic(哥特体)等",
						Type:        "string",
					},
				},
				Required: []string{"text"},
			},
		},
	}

	reqBody := SFRequest{
		Model:       h.Model,
		Messages:    h.history,
		MaxTokens:   512,
		Stream:      true, // Enable streaming
		Temperature: 0.7,
		Tools:       tools,
	}
	body, _ := json.Marshal(reqBody)

	// Generate unique playID
	playID := fmt.Sprintf("sf-%s", uuid.New().String())
	h.logger.WithField("playID", playID).Info("Starting SiliconFlow stream with playID")

	req, _ := http.NewRequestWithContext(h.ctx, "POST", h.Endpoint, bytes.NewReader(body))
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", "Bearer "+h.APIKey)
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	reader := bufio.NewReader(resp.Body)
	var buffer string
	fullResponse := ""
	var shouldHangup bool
	// shane: tooCallsMap
	toolCallsMap := make(map[string]struct {
		Id       string
		Type     string
		Function struct {
			Name      string
			Arguments string
		}
	})
	// shane: record the order
	var toolCallOrder []string

	// Regular expression to detect punctuation
	punctuationRegex := regexp.MustCompile(`([.,;:!?，。！？；：])\s*`)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("error reading stream: %w", err)
		}

		lineStr := strings.TrimSpace(string(line))
		if lineStr == "" || !strings.HasPrefix(lineStr, "data: ") {
			continue
		}

		data := strings.TrimPrefix(lineStr, "data: ")
		if data == "[DONE]" {
			break
		}

		var respData RespData
		if err := json.Unmarshal([]byte(data), &respData); err != nil {
			continue
		}

		// Process content if available
		if len(respData.Choices) > 0 {
			choice := respData.Choices[0]
			// shane: process content
			if choice.Delta.Content != "" {
				content := choice.Delta.Content
				buffer += content
				fullResponse += content

				// Check for punctuation in the buffer
				matches := punctuationRegex.FindAllStringSubmatchIndex(buffer, -1)
				if len(matches) > 0 {
					lastIdx := 0
					for _, match := range matches {
						segment := buffer[lastIdx:match[1]]
						if segment != "" {
							if err := ttsCallback(segment, playID, false); err != nil {
								h.logger.WithError(err).Error("Failed to send TTS segment")
							}
						}
						lastIdx = match[1]
					}
					if lastIdx < len(buffer) {
						buffer = buffer[lastIdx:]
					} else {
						buffer = ""
					}
				}
			}

			// shane: construct tool calls
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if tc.Id != "" {
						if tool, ok := toolCallsMap[tc.Id]; ok {
							if tc.Function.Name != "" {
								tool.Function.Name = tc.Function.Name
							}
							if tc.Function.Arguments != "" {
								tool.Function.Arguments += tc.Function.Arguments
							}
							toolCallsMap[tc.Id] = tool
						} else {
							toolCallsMap[tc.Id] = struct {
								Id       string
								Type     string
								Function struct {
									Name      string
									Arguments string
								}
							}{
								Id:   tc.Id,
								Type: tc.Type,
								Function: struct {
									Name      string
									Arguments string
								}{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							}
							toolCallOrder = append(toolCallOrder, tc.Id)
						}
					} else {
						if tc.Function.Arguments != "" && len(toolCallOrder) > 0 {
							lastToolId := toolCallOrder[len(toolCallOrder)-1]
							if tool, ok := toolCallsMap[lastToolId]; ok {
								tool.Function.Arguments += tc.Function.Arguments
								toolCallsMap[lastToolId] = tool
							}
						}
					}
				}
			}
		}
	}

	// shane: Send any remaining text in the buffer
	if err := ttsCallback(buffer, playID, shouldHangup); err != nil {
		h.logger.WithError(err).Error("Failed to send final TTS segment")
	}

	// shane: Add assistant's response to history
	h.history = append(h.history, SFMessage{
		Role:    "assistant",
		Content: fullResponse,
	})

	// shane:
	for _, toolCall := range toolCallsMap {
		if toolCall.Id != "" && toolCall.Function.Name != "" && toolCall.Function.Arguments != "" {
			thinkingMsg := "正在思考，请稍等片刻"
			if err := ttsCallback(thinkingMsg, playID, false); err != nil {
				h.logger.WithError(err).Error("Failed to send thinking TTS")
			}

			// shane: 异步
			go func(tc struct {
				Id       string
				Type     string
				Function struct {
					Name      string
					Arguments string
				}
			}) {
				var result string
				var err error

				switch tc.Function.Name {
				case "queryWeather":
					h.logger.Info("[Handling queryWeather tool call in stream]")
					result, err = h.handleQueryWeather(tc.Function.Arguments, userMsg, tc.Id)
				case "searchOnline":
					h.logger.Info("[Handling searchOnline tool call in stream]")
					result, err = h.handleSearchOnline(tc.Function.Arguments, userMsg, tc.Id)
				case "generateImage":
					h.logger.Info("[Handling generateImage tool call in stream]")
					result, err = h.handleGenerateImage(tc.Function.Arguments)
				case "transformText":
					logrus.Info("[Handling transformText tool call]")
					result, err = h.handleTransformText(tc.Function.Arguments, userMsg, tc.Id)
				default:
					err = fmt.Errorf("unknown function: %s", tc.Function.Name)
				}

				if err != nil {
					h.logger.WithError(err).Error("Tool call failed")
					ttsCallback("查询失败，请稍后重试", playID, false)
				} else {
					ttsCallback(result, playID, false)
				}
			}(toolCall)
		}
	}

	h.logger.WithFields(logrus.Fields{
		"responseLength": len(fullResponse),
		"hangup":         shouldHangup,
	}).Info("SiliconFlow stream completed")

	return fullResponse, nil
}

func (h *SiliconFlowHandler) Query(userMsg string) (string, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.history = append(h.history, SFMessage{
		Role:    "user",
		Content: userMsg,
	})

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
				Description: "查询指定地点的天气信息,地点必须是中文，不能是英文！需要剥离出省份信息放到sheng参数里面,并且需要剥离出地点的信息放到place参数里面",
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
	}

	reqBody := SFRequest{
		Model:     h.Model,
		Messages:  h.history,
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

	if len(sfResp.Choices) > 0 {
		assistantMsg := sfResp.Choices[0].Message
		h.history = append(h.history, SFMessage{
			Role:    "assistant",
			Content: assistantMsg.Content,
		})
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
	Chan := make(chan struct {
		result SearchOnlineStruct
		err    error
	}, 1)
	// shane: 优化为协程
	go func() {
		result, err := h.SearchOnline(argumentsJson.Query)
		Chan <- struct {
			result SearchOnlineStruct
			err    error
		}{result, err}
	}()
	// shane: wait for the search result
	searchResp := <-Chan
	if searchResp.err != nil {
		h.logger.Errorf("Failed to search online arguments: %v", searchResp.err)
		return "", searchResp.err
	}

	searchOnline := searchResp.result
	if len(searchOnline.Choices) == 0 || len(searchOnline.Choices[0].Message.ToolCalls) == 0 {
		return "", fmt.Errorf("no search results found")
	}
	searchResult := searchOnline.Choices[0].Message.ToolCalls[0].SearchResult

	return h.searchResultToString(searchResult), nil
	// shane:
	// h.mutex.Lock()
	// reqBody := SFRequest{
	// 	Model: h.Model,
	// 	Messages: append(h.history,
	// 		SFMessage{
	// 			Role: "tool", Content: h.searchResultToString(searchResult), ToolCallId: toolCallId,
	// 		},
	// 	),
	// 	MaxTokens: 512,
	// }
	// h.mutex.Unlock()
	// body, _ := json.Marshal(reqBody)

	// req, _ := http.NewRequestWithContext(h.ctx, "POST", h.Endpoint, bytes.NewReader(body))
	// req.Header.Set("accept", "application/json")
	// req.Header.Set("authorization", "Bearer "+h.APIKey)
	// req.Header.Set("content-type", "application/json")

	// resp, err := http.DefaultClient.Do(req)
	// if err != nil {
	// 	return "", err
	// }
	// defer resp.Body.Close()
	// respBody, _ := ioutil.ReadAll(resp.Body)

	// if resp.StatusCode != 200 {
	// 	return "", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(respBody))
	// }

	// var sfResp SFResponse
	// if err := json.Unmarshal(respBody, &sfResp); err != nil {
	// 	return "", err
	// }
	// if len(sfResp.Choices) == 0 {
	// 	return "", fmt.Errorf("no choices in response")
	// }
	// // shane: append history
	// h.mutex.Lock()
	// if len(sfResp.Choices) > 0 {
	// 	h.history = append(h.history, SFMessage{
	// 		Role:    "assistant",
	// 		Content: sfResp.Choices[0].Message.Content,
	// 	})
	// }
	// h.mutex.Unlock()
	// return sfResp.Choices[0].Message.Content, nil
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
										Content: fmt.Sprintf("%s", cleanStr),
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
	Chan := make(chan struct {
		response WeatherResponse
		err      error
	}, 1)
	// shane: 优化为协程
	go func() {
		// shane: construct the request data
		data := fmt.Sprintf("id=10006512&key=512b69d6b44c1c59a1a698da8d3cb1a7&sheng=%s&place=%s", args.Sheng, args.Place)
		request, err := http.NewRequest("POST", "https://cn.apihz.cn/api/tianqi/tqyb.php", bytes.NewBufferString(data))
		if err != nil {
			Chan <- struct {
				response WeatherResponse
				err      error
			}{WeatherResponse{}, err}
			return
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			Chan <- struct {
				response WeatherResponse
				err      error
			}{WeatherResponse{}, err}
			return
		}
		defer response.Body.Close()

		respBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			Chan <- struct {
				response WeatherResponse
				err      error
			}{WeatherResponse{}, err}
			return
		}

		var weatherResp WeatherResponse
		if err := json.Unmarshal(respBody, &weatherResp); err != nil {
			Chan <- struct {
				response WeatherResponse
				err      error
			}{WeatherResponse{}, fmt.Errorf("failed to unmarshal weather response: %v, content: %s", err, string(respBody))}
			return
		}

		// shane: send weatherResponse to chan
		Chan <- struct {
			response WeatherResponse
			err      error
		}{weatherResp, nil}
	}()
	// shane: construct the weather information string
	weatherResp := <-Chan
	weatherInfo := fmt.Sprintf("当前时间：%s,%s 的天气情况如下：温度为 %.1f 摄氏度，天气状况为 %s，风力为 %s，相对湿度为 %d%%。",
		weatherResp.response.Uptime, weatherResp.response.Place, weatherResp.response.Temperature, weatherResp.response.Weather1, weatherResp.response.WindScale, weatherResp.response.Humidity)

	return weatherInfo, nil

	//h.mutex.Lock()
	//reqBody := SFRequest{
	//	Model: h.Model,
	//	Messages: append(h.history, SFMessage{
	//		Role: "tool", Content: weatherInfo, ToolCallId: toolCallId,
	//	}),
	//	MaxTokens: 512,
	//}
	//h.mutex.Unlock()
	//body, _ := json.Marshal(reqBody)
	//
	//req, _ := http.NewRequestWithContext(h.ctx, "POST", h.Endpoint, bytes.NewReader(body))
	//req.Header.Set("accept", "application/json")
	//req.Header.Set("authorization", "Bearer "+h.APIKey)
	//req.Header.Set("content-type", "application/json")
	//
	//resp, err := http.DefaultClient.Do(req)
	//if err != nil {
	//	return "", err
	//}
	//defer resp.Body.Close()
	//respBody, _ := ioutil.ReadAll(resp.Body)
	//
	//if resp.StatusCode != 200 {
	//	return "", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(respBody))
	//}
	//
	//var sfResp SFResponse
	//if err := json.Unmarshal(respBody, &sfResp); err != nil {
	//	return "", err
	//}
	//if len(sfResp.Choices) == 0 {
	//	return "", fmt.Errorf("no choices in response")
	//}
	//// shane: append history
	//h.mutex.Lock()
	//if len(sfResp.Choices) > 0 {
	//	h.history = append(h.history, SFMessage{
	//		Role:    "assistant",
	//		Content: sfResp.Choices[0].Message.Content,
	//	})
	//}
	//h.mutex.Unlock()
	//return sfResp.Choices[0].Message.Content, nil
}

// TextTransform
func (h *SiliconFlowHandler) handleTransformText(arguments string, userMsg string, toolCallId string) (string, error) {
	args := struct {
		Text   string `json:"text"`
		Prompt string `json:"prompt"`
		Style  string `json:"style"`
	}{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		h.logger.Errorf("Failed to unmarshal transform text arguments: %v", err)
		return "", err
	}

	if args.Prompt == "" {
		return "", fmt.Errorf("prompt is required and cannot be empty")
	}

	// Ine: Constructing the request body
	transformReq := TextTransformRequest{
		Model: "wordart-semantic",
		Input: struct {
			Text   string `json:"text"`
			Prompt string `json:"prompt"`
		}{
			Text:   args.Text,
			Prompt: h.getStylePrompt(args.Prompt),
		},
		Parameters: struct {
			Steps            int    `json:"steps"`
			N                int    `json:"n"`
			FontName         string `json:"font_name,omitempty"`
			TtfUrl           string `json:"ttf_url,omitempty"`
			OutputImageRatio string `json:"output_image_ratio"`
		}{
			Steps:            60,
			N:                2,
			OutputImageRatio: "1280x720",
			FontName:         h.getFontName(args.Style),
			TtfUrl:           h.getTtfUrl(args.Style),
		},
	}

	marshal, err := json.Marshal(transformReq)
	if err != nil {
		h.logger.Errorf("Failed to marshal transform text request: %v", err)
		return "", err
	}

	// Ine: Creating HTTP Requests
	request, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/wordart/semantic", bytes.NewReader(marshal))
	if err != nil {
		h.logger.Errorf("Failed to create transform text request: %v", err)
		return "", err
	}

	request.Header.Set("X-DashScope-Async", "enable")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer sk-2fc7c45d76494b279e4e029536616140")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		h.logger.Errorf("Failed to send transform text request: %v", err)
		return "", err
	}
	defer response.Body.Close()

	resp, err := ioutil.ReadAll(response.Body)
	if err != nil {
		h.logger.Errorf("Failed to read transform text response: %v", err)
		return "", err
	}

	h.logger.Infof("Transform text API response: %s", string(resp))

	if response.StatusCode != 200 {
		h.logger.Errorf("Transform text API returned status code: %d, body: %s", response.StatusCode, string(resp))
		return "", fmt.Errorf("transform text API error: status code %d, body: %s", response.StatusCode, string(resp))
	}

	var TFResponse TextTransformResponse
	if err := json.Unmarshal(resp, &TFResponse); err != nil {
		h.logger.Errorf("Failed to unmarshal transform text response: %v, content: %s", err, string(resp))
		return "", fmt.Errorf("failed to unmarshal transform text response: %v", err)
	}

	// Ine: Get Task ID
	taskId := TFResponse.Output.TaskId
	if taskId == "" {
		return "", fmt.Errorf("no task_id returned from API")
	}

	TFResult, err := h.queryTaskResult(taskId)
	if err != nil {
		return "", fmt.Errorf("failed to query task result: %v", err)
	}

	return TFResult, nil

	// // Ine: Calling LLM to process results
	// reqBody := SFRequest{
	// 	Model: h.Model,
	// 	Messages: append(h.history, SFMessage{
	// 		Role:       "tool",
	// 		Content:    TFResult,
	// 		ToolCallId: toolCallId,
	// 	}),
	// 	MaxTokens: 512,
	// }

	// body, _ := json.Marshal(reqBody)

	// // Ine: Sending LLM requests
	// req, _ := http.NewRequestWithContext(h.ctx, "POST", h.Endpoint, bytes.NewReader(body))
	// req.Header.Set("accept", "application/json")
	// req.Header.Set("authorization", "Bearer "+h.APIKey)
	// req.Header.Set("content-type", "application/json")

	// llmResp, err := http.DefaultClient.Do(req)
	// if err != nil {
	// 	return "", err
	// }
	// defer llmResp.Body.Close()

	// llmRespBody, _ := ioutil.ReadAll(llmResp.Body)

	// if llmResp.StatusCode != 200 {
	// 	return "", fmt.Errorf("status code: %d, body: %s", llmResp.StatusCode, string(llmRespBody))
	// }

	// var sfResp SFResponse
	// if err := json.Unmarshal(llmRespBody, &sfResp); err != nil {
	// 	return "", err
	// }
	// if len(sfResp.Choices) == 0 {
	// 	return "", fmt.Errorf("no choices in response")
	// }

	// // Ine: Adding to history
	// if len(sfResp.Choices) > 0 {
	// 	h.history = append(h.history, SFMessage{
	// 		Role:    "assistant",
	// 		Content: sfResp.Choices[0].Message.Content,
	// 	})
	// }

	// return sfResp.Choices[0].Message.Content, nil
}

// Ine: Added a method to query task results
func (h *SiliconFlowHandler) queryTaskResult(taskId string) (string, error) {
	queryUrl := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/tasks/%s", taskId)

	// Ine: Polling query
	maxAttempts := 30
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Create a query request
		request, err := http.NewRequest("GET", queryUrl, nil)
		if err != nil {
			return "", err
		}

		request.Header.Set("Authorization", "Bearer sk-2fc7c45d76494b279e4e029536616140")

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			return "", err
		}

		resp, err := ioutil.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			return "", err
		}

		if response.StatusCode != 200 {
			return "", fmt.Errorf("query task API error: status code %d, body: %s", response.StatusCode, string(resp))
		}

		var queryResp TaskQueryResponse
		if err := json.Unmarshal(resp, &queryResp); err != nil {
			return "", fmt.Errorf("failed to unmarshal query response: %v", err)
		}

		h.logger.Infof("Task %s status: %s", taskId, queryResp.Output.TaskStatus)

		switch queryResp.Output.TaskStatus {
		case "SUCCEEDED":
			// resultStr := fmt.Sprintf("艺术字生成成功！生成了%d张图片：", len(queryResp.Output.Results))
			// for _, result := range queryResp.Output.Results {
			// 	resultStr += fmt.Sprintf("%s", result.PngUrl)
			// }
			resultStr := fmt.Sprintf("![艺术字](%s)", queryResp.Output.Results[0].PngUrl)
			return resultStr, nil
		case "FAILED":
			return "", fmt.Errorf("task failed")
		case "PENDING", "RUNNING":
			// The task is still being processed. Please wait for 10 seconds and try again.
			time.Sleep(10 * time.Second)
			continue
		default:
			return "", fmt.Errorf("unknown task status: %s", queryResp.Output.TaskStatus)
		}
	}

	return "", fmt.Errorf("task timeout after %d attempts", maxAttempts)
}

// Ine: Add prompt word processing
func (h *SiliconFlowHandler) getStylePrompt(userPrompt string) string {
	return userPrompt
}

// Ine: Font-Style
func (h *SiliconFlowHandler) getFontName(style string) string {
	// Preset
	fontMap := map[string]string{
		"dongfangdakai":  "dongfangdakai",    // 阿里妈妈东方大楷
		"puhuiti":        "puhuiti_m",        // 阿里巴巴普惠体
		"shuheiti":       "shuheiti",         // 阿里妈妈数黑体
		"jinbu":          "jinbu1",           // 钉钉进步体
		"kuhei":          "kuheti1",          // 站酷酷黑体
		"kuailei":        "kuailei1",         // 站酷快乐体
		"wenyiti":        "wenyiti1",         // 站酷文艺体
		"logoti":         "logoti",           // 站酷小薇LOGO体
		"cangeryuyangti": "cangeryuyangti_m", // 站酷仓耳渔阳体
		"siyuansongti":   "siyuansongti_b",   // 思源宋体
		"siyuanheiti":    "siyuanheiti_m",    // 思源黑体
		"fangzhengkaiti": "fangzhengkaiti",   // 方正楷体
		"flower":         "dongfangdakai",    // 花体样式默认使用东方大楷
		"art":            "puhuiti_m",        // 艺术字样式默认使用普惠体
		"gothic":         "siyuanheiti_m",    // 哥特样式默认使用思源黑体
		"modern":         "siyuansongti_b",   // 现代样式默认使用思源宋体
	}

	if fontName, exists := fontMap[style]; exists {
		return fontName
	}
	return "dongfangdakai" // 默认使用东方大楷
}

// Ine: URL of the custom font TTF file
func (h *SiliconFlowHandler) getTtfUrl(style string) string {
	// Ine：ttf_url & font_name cannot be used at the same time
	customFontMap := map[string]string{
		"custom_font1": "https://example.com/fonts/custom1.ttf",
		"custom_font2": "https://example.com/fonts/custom2.ttf",
		// Add more...
	}

	if url, exists := customFontMap[style]; exists {
		return url
	}
	return "" // 大部分情况下使用预设字体，返回空字符串
}

// shane: struct2string
func (h *SiliconFlowHandler) searchResultToString(results []SearchResult) string {
	var buf bytes.Buffer
	for _, r := range results {
		buf.WriteString(fmt.Sprintf("Title: %s\nContent: %s\nLink: %s\n\n", r.Title, r.Content, r.Link))
	}
	return buf.String()
}

func (h *SiliconFlowHandler) ResetHistory() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.history = []SFMessage{}
}
