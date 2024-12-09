package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/riza-io/mcp-go/internal/jsonrpc"
)

type Server interface {
	Initialize(ctx context.Context, req *Request[InitializeRequest]) (*Response[InitializeResponse], error)
	ListTools(ctx context.Context, req *Request[ListToolsRequest]) (*Response[ListToolsResponse], error)
	CallTool(ctx context.Context, req *Request[CallToolRequest]) (*Response[CallToolResponse], error)
	ListPrompts(ctx context.Context, req *Request[ListPromptsRequest]) (*Response[ListPromptsResponse], error)
	GetPrompt(ctx context.Context, req *Request[GetPromptRequest]) (*Response[GetPromptResponse], error)
	ListResources(ctx context.Context, req *Request[ListResourcesRequest]) (*Response[ListResourcesResponse], error)
	ReadResource(ctx context.Context, req *Request[ReadResourceRequest]) (*Response[ReadResourceResponse], error)
	ListResourceTemplates(ctx context.Context, req *Request[ListResourceTemplatesRequest]) (*Response[ListResourceTemplatesResponse], error)
	Completion(ctx context.Context, req *Request[CompletionRequest]) (*Response[CompletionResponse], error)
}

type UnimplementedServer struct{}

func (s *UnimplementedServer) Initialize(ctx context.Context, req *Request[InitializeRequest]) (*Response[InitializeResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func (s *UnimplementedServer) ListTools(ctx context.Context, req *Request[ListToolsRequest]) (*Response[ListToolsResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func (s *UnimplementedServer) CallTool(ctx context.Context, req *Request[CallToolRequest]) (*Response[CallToolResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func (s *UnimplementedServer) ListPrompts(ctx context.Context, req *Request[ListPromptsRequest]) (*Response[ListPromptsResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func (s *UnimplementedServer) GetPrompt(ctx context.Context, req *Request[GetPromptRequest]) (*Response[GetPromptResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func (s *UnimplementedServer) ListResources(ctx context.Context, req *Request[ListResourcesRequest]) (*Response[ListResourcesResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func (s *UnimplementedServer) ReadResource(ctx context.Context, req *Request[ReadResourceRequest]) (*Response[ReadResourceResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func (s *UnimplementedServer) ListResourceTemplates(ctx context.Context, req *Request[ListResourceTemplatesRequest]) (*Response[ListResourceTemplatesResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func (s *UnimplementedServer) Completion(ctx context.Context, req *Request[CompletionRequest]) (*Response[CompletionResponse], error) {
	return nil, fmt.Errorf("unimplemented")
}

func process[T, V any](ctx context.Context, msg jsonrpc.Request, params *T, method func(ctx context.Context, req *Request[T]) (*Response[V], error)) (any, error) {
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return nil, err
	}
	req := NewRequest(params)
	req.id = msg.ID.String()
	resp, rerr := method(ctx, req)
	if rerr != nil {
		return nil, rerr
	}
	return resp.Result, nil
}

func Listen(ctx context.Context, r io.Reader, w io.Writer, logger *slog.Logger, srv Server) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		dec := json.NewDecoder(strings.NewReader(line))

		var msg jsonrpc.Request
		if err := dec.Decode(&msg); err != nil {
			logger.Error("decode", "err", err)
			continue
		}

		var result any
		var err error

		switch msg.Method {
		case "initialize":
			params := &InitializeRequest{}
			result, err = process(ctx, msg, params, srv.Initialize)
		case "completion/complete":
			params := &CompletionRequest{}
			result, err = process(ctx, msg, params, srv.Completion)
		case "tools/list":
			params := &ListToolsRequest{}
			result, err = process(ctx, msg, params, srv.ListTools)
		case "tools/call":
			params := &CallToolRequest{}
			result, err = process(ctx, msg, params, srv.CallTool)
		case "prompts/list":
			params := &ListPromptsRequest{}
			result, err = process(ctx, msg, params, srv.ListPrompts)
		case "prompts/get":
			params := &GetPromptRequest{}
			result, err = process(ctx, msg, params, srv.GetPrompt)
		case "resources/list":
			params := &ListResourcesRequest{}
			result, err = process(ctx, msg, params, srv.ListResources)
		case "resources/read":
			params := &ReadResourceRequest{}
			result, err = process(ctx, msg, params, srv.ReadResource)
		case "resources/templates/list":
			params := &ListResourceTemplatesRequest{}
			result, err = process(ctx, msg, params, srv.ListResourceTemplates)
		default:
			if msg.ID == "" {
				// Ignore notifications
				continue
			}
			err = fmt.Errorf("unsupported method: %s", msg.Method)
		}

		logger.Info("rpc",
			"method", msg.Method,
			"params", msg.Params,
			"result", result,
			"error", err,
		)

		var resp any
		if err != nil {
			resp = jsonrpc.Error[any]{
				ID:      msg.ID,
				JsonRPC: msg.JsonRPC,
				Error: jsonrpc.ErrorDetail[any]{
					Code:    1,
					Message: err.Error(),
				},
			}
		} else {
			resp = jsonrpc.Response[any]{
				ID:      msg.ID,
				JsonRPC: msg.JsonRPC,
				Result:  result,
			}
		}

		bs, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		logger.Info("rpc", "response", string(bs))
		fmt.Fprintln(w, string(bs))
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scan", "err", err)
		return err
	}
	return nil
}