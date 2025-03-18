package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/fanchunke/deeppick-ai/internal/config"
	"github.com/fanchunke/deeppick-ai/internal/repository"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/labstack/echo/v4"
	"github.com/openai/openai-go"
	"github.com/panjf2000/ants/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type TaskStatus string

const (
	Pending TaskStatus = "pending"
	Running TaskStatus = "running"
	Success TaskStatus = "success"
	Failed  TaskStatus = "failed"
)

type DetectionService struct {
	client *openai.Client
	cfg    *config.Config
	tracer trace.Tracer
	db     *repository.Queries
	pool   *ants.Pool
}

func NewChatCompletionService(client *openai.Client, cfg *config.Config, db *sql.DB, pool *ants.Pool) *DetectionService {
	return &DetectionService{client: client, cfg: cfg, tracer: otel.Tracer("DetectionService"), db: repository.New(db)}
}

type DetectionType string

const (
	FruitDetection     DetectionType = "fruit"
	VegetableDetection DetectionType = "vegetable"
)

type DetectImageRequest struct {
	ImageUrl      string        `json:"image_url"`
	DetectionType DetectionType `json:"detection_type"`
}

type DetectImageResponse struct {
	Name           string       `json:"name" jsonschema_description:"The object's name detected in the image"`
	ScientificName string       `json:"scientific_name" jsonschema_description:"The object's scientific_name detected in the image"`
	Category       string       `json:"category" jsonschema_description:"The object's category detected in the image"`
	Family         string       `json:"family" jsonschema_description:"The object's family detected in the image"`
	Metrics        []Metric     `json:"metrics" jsonschema_description:"The object's metrics detected in the image"`
	OverallScore   OverallScore `json:"overall_score" jsonschema_description:"The object's overall_score detected in the image"`
	ExpertAdvice   ExpertAdvice `json:"expert_advice" jsonschema_description:"The object's expert_advice detected in the image"`
}

type ExpertAdvice struct {
	Storage   string `json:"storage" jsonschema_description:"Expert's storage advice of the object detected in the image"`
	Nutrition string `json:"nutrition" jsonschema_description:"The object's nutrition detected in the image"`
	Selection string `json:"selection" jsonschema_description:"Expert's selection advice of the object detected in the image"`
}

type Metric struct {
	Name  string  `json:"name" jsonschema_description:"The metric English name of the object to judgment"`
	Label string  `json:"label" jsonschema_description:"The metric Chinese label name of the object to judgment"`
	Value float64 `json:"value" jsonschema_description:"The metric score of the object to judgment"`
	Basis string  `json:"basis" jsonschema_description:"judgment basis of the metric value"`
}

type OverallScore struct {
	Score  float64 `json:"score" jsonschema_description:"Overall score of the object detected in the image based on the metrics"`
	Reason string  `json:"reason" jsonschema_description:"Judgment reason of the overall score"`
}

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

var DetectImageResponseSchema = GenerateSchema[DetectImageResponse]()

type DetectionTaskResponse struct {
	TaskId string `json:"task_id"`
}

func (s *DetectionService) DetectImage() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var req DetectImageRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}

		taskId := uuid.New().String()
		if _, err := s.db.CreateTask(ctx, repository.CreateTaskParams{TaskID: taskId, Status: string(Pending)}); err != nil {
			return err
		}

		newCtx := trace.ContextWithSpan(context.Background(), trace.SpanFromContext(ctx))
		if err := s.pool.Submit(func() {
			if _, err := s.detectImage(newCtx, &req, taskId); err != nil {
				log.Printf("exec detection task %s failed: %#v", taskId, err)
			}
		}); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, DetectionTaskResponse{TaskId: taskId})
	}
}

func (s *DetectionService) detectImage(ctx context.Context, req *DetectImageRequest, taskId string) (*DetectImageResponse, error) {
	// 更新任务状态
	if _, err := s.db.UpdateTaskStatus(ctx, repository.UpdateTaskStatusParams{
		TaskID: taskId,
		Status: string(Running),
	}); err != nil {
		return nil, err
	}

	// 开始检测
	schema := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("ImageDetectResult"),
		Description: openai.F("image detect result"),
		Schema:      openai.F(DetectImageResponseSchema),
		Strict:      openai.Bool(true),
	}
	ctx, span := s.tracer.Start(ctx, "chatCompletion")
	chatCompletion, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(FruitAndVegetableDetectionPrompt),
			openai.UserMessage("帮我识别，返回json"),
			openai.UserMessageParts(openai.ImagePart(req.ImageUrl)),
		}),
		Model: openai.F(openai.ChatModel(s.cfg.OpenAI.Model)),
		ResponseFormat: openai.F(openai.ChatCompletionNewParamsResponseFormatUnion(
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schema),
			},
		)),
	})
	if err != nil {
		if _, err := s.db.UpdateTaskStatus(ctx, repository.UpdateTaskStatusParams{
			TaskID: taskId,
			Status: string(Failed),
		}); err != nil {
			return nil, err
		}
		return nil, err
	}
	span.End()

	if len(chatCompletion.Choices) == 0 {
		if _, err := s.db.UpdateTaskStatus(ctx, repository.UpdateTaskStatusParams{
			TaskID: taskId,
			Status: string(Failed),
		}); err != nil {
			return nil, err
		}
		return nil, errors.New("大模型无返回结果")
	}

	if _, err := s.db.UpdateTaskResult(ctx, repository.UpdateTaskResultParams{
		TaskID: taskId,
		Status: string(Success),
		Result: json.RawMessage([]byte(chatCompletion.Choices[0].Message.Content)),
	}); err != nil {
		return nil, err
	}

	var response DetectImageResponse
	if err := json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

type GetTaskRequest struct {
	TaskId string `param:"task_id"`
}

func (s *DetectionService) GetTask() echo.HandlerFunc {
	return func(c echo.Context) error {
		var req GetTaskRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}

		ctx := c.Request().Context()
		result, err := s.db.GetTask(ctx, req.TaskId)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, result)
	}
}
