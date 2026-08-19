package localgrpc

import (
	"context"
	"fmt"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var _ grpc.ServiceRegistrar = (*Conn)(nil)
var _ grpc.ClientConnInterface = (*Conn)(nil)

type unaryMethod struct {
	description *grpc.MethodDesc
	service     any
}

type streamMethod struct {
	description *grpc.StreamDesc
	service     any
}

// ResponseMetadata 保存进程内 unary gRPC 调用返回的响应元数据。
type ResponseMetadata struct {
	// Header 是服务端响应头。
	Header metadata.MD
	// Trailer 是服务端响应尾部元数据。
	Trailer metadata.MD
}

type responseMetadataContextKey struct{}

// WithResponseMetadata 将响应元数据载体写入进程内调用上下文。
func WithResponseMetadata(ctx context.Context, response *ResponseMetadata) context.Context {
	return context.WithValue(ctx, responseMetadataContextKey{}, response)
}

// ResponseMetadataFromContext 从进程内调用上下文读取响应元数据载体。
func ResponseMetadataFromContext(ctx context.Context) *ResponseMetadata {
	if ctx == nil {
		return nil
	}
	response, _ := ctx.Value(responseMetadataContextKey{}).(*ResponseMetadata)
	return response
}

// Conn 将注册的 gRPC 服务作为进程内生成客户端连接使用。
type Conn struct {
	mu                       sync.RWMutex
	methods                  map[string]unaryMethod
	streams                  map[string]streamMethod
	interceptor              grpc.UnaryServerInterceptor
	streamInterceptor        grpc.StreamServerInterceptor
	clientInterceptors       []grpc.UnaryClientInterceptor
	streamClientInterceptors []grpc.StreamClientInterceptor
}

// Option 配置进程内 gRPC 连接。
type Option func(*Conn)

// WithUnaryInterceptor 设置进程内调用使用的 unary 拦截器。
func WithUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) Option {
	return func(conn *Conn) { conn.interceptor = interceptor }
}

// WithStreamInterceptor 设置进程内调用使用的 stream 拦截器。
func WithStreamInterceptor(interceptor grpc.StreamServerInterceptor) Option {
	return func(conn *Conn) { conn.streamInterceptor = interceptor }
}

// WithUnaryClientInterceptor 追加进程内 unary 客户端拦截器。
func WithUnaryClientInterceptor(interceptors ...grpc.UnaryClientInterceptor) Option {
	return func(conn *Conn) {
		conn.clientInterceptors = append(conn.clientInterceptors, interceptors...)
	}
}

// WithStreamClientInterceptor 追加进程内 stream 客户端拦截器。
func WithStreamClientInterceptor(interceptors ...grpc.StreamClientInterceptor) Option {
	return func(conn *Conn) {
		conn.streamClientInterceptors = append(conn.streamClientInterceptors, interceptors...)
	}
}

// NewConn 创建进程内 gRPC 连接。
func NewConn(options ...Option) *Conn {
	conn := &Conn{methods: make(map[string]unaryMethod), streams: make(map[string]streamMethod)}
	for _, option := range options {
		if option != nil {
			option(conn)
		}
	}
	return conn
}

// RegisterService 注册生成代码声明的 gRPC 服务。
func (c *Conn) RegisterService(description *grpc.ServiceDesc, service any) {
	if description == nil {
		panic("localgrpc: gRPC 服务描述不能为空")
	}
	if service == nil {
		panic(fmt.Sprintf("localgrpc: gRPC 服务 %s 的实现不能为空", description.ServiceName))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for index := range description.Methods {
		method := &description.Methods[index]
		c.registerUnary(description.ServiceName, method, service)
	}
	for index := range description.Streams {
		stream := &description.Streams[index]
		fullMethod := "/" + description.ServiceName + "/" + stream.StreamName
		if _, exists := c.methods[fullMethod]; exists {
			panic(fmt.Sprintf("localgrpc: gRPC 方法 %s 重复注册", fullMethod))
		}
		if _, exists := c.streams[fullMethod]; exists {
			panic(fmt.Sprintf("localgrpc: gRPC 方法 %s 重复注册", fullMethod))
		}
		c.streams[fullMethod] = streamMethod{description: stream, service: service}
	}
}

// Invoke 调用已注册的进程内 unary gRPC 方法。
func (c *Conn) Invoke(ctx context.Context, method string, args, reply any, options ...grpc.CallOption) error {
	responseMetadata := new(ResponseMetadata)
	ctx = WithResponseMetadata(ctx, responseMetadata)
	invoker := func(ctx context.Context, method string, req, response any, _ *grpc.ClientConn, callOptions ...grpc.CallOption) error {
		return c.invoke(ctx, method, req, response)
	}
	for index := len(c.clientInterceptors) - 1; index >= 0; index-- {
		interceptor := c.clientInterceptors[index]
		if interceptor == nil {
			continue
		}
		next := invoker
		invoker = func(ctx context.Context, method string, req, response any, conn *grpc.ClientConn, callOptions ...grpc.CallOption) error {
			return interceptor(ctx, method, req, response, conn, next, callOptions...)
		}
	}
	err := invoker(ctx, method, args, reply, nil, options...)
	applyCallOptions(options, responseMetadata)
	return err
}

// invoke 执行 unary 调用并由 Connection 层负责应用 CallOption。
func (c *Conn) invoke(ctx context.Context, method string, args, reply any) error {
	c.mu.RLock()
	registered, exists := c.methods[method]
	interceptor := c.interceptor
	c.mu.RUnlock()
	if !exists {
		return status.Errorf(codes.Unimplemented, "进程内 gRPC 方法未注册: %s", method)
	}
	var response any
	var err error
	func() {
		defer func() {
			if panicValue := recover(); panicValue != nil {
				err = fmt.Errorf("进程内 gRPC 方法处理器异常: %v", panicValue)
			}
		}()
		response, err = registered.description.Handler(registered.service, ctx, func(request any) error {
			return copyMessage(request, args)
		}, interceptor)
	}()
	if err != nil {
		return err
	}
	return copyMessage(reply, response)
}

// NewStream 创建可直接调用本地服务处理器的进程内 streaming RPC。
func (c *Conn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, options ...grpc.CallOption) (grpc.ClientStream, error) {
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, conn *grpc.ClientConn, method string, callOptions ...grpc.CallOption) (grpc.ClientStream, error) {
		return c.newStream(ctx, method, callOptions...)
	}
	for index := len(c.streamClientInterceptors) - 1; index >= 0; index-- {
		interceptor := c.streamClientInterceptors[index]
		if interceptor == nil {
			continue
		}
		next := streamer
		streamer = func(ctx context.Context, desc *grpc.StreamDesc, conn *grpc.ClientConn, method string, callOptions ...grpc.CallOption) (grpc.ClientStream, error) {
			return interceptor(ctx, desc, conn, method, next, callOptions...)
		}
	}
	return streamer(ctx, desc, nil, method, options...)
}

// newStream 创建并启动本地 stream 调用。
func (c *Conn) newStream(ctx context.Context, method string, options ...grpc.CallOption) (grpc.ClientStream, error) {
	c.mu.RLock()
	registered, exists := c.streams[method]
	c.mu.RUnlock()
	if !exists {
		return nil, status.Errorf(codes.Unimplemented, "进程内 gRPC 流式方法未注册: %s", method)
	}
	stream := newLocalStream(ctx, registered.description, method)
	interceptor := c.streamInterceptor
	clientStream := &localClientStream{localStream: stream}
	clientStream.configureCallOptions(options)
	go func() {
		var err error
		func() {
			defer func() {
				if panicValue := recover(); panicValue != nil {
					err = fmt.Errorf("进程内 gRPC 流式处理器异常: %v", panicValue)
				}
			}()
			if interceptor == nil {
				err = registered.description.Handler(registered.service, stream)
				return
			}
			info := &grpc.StreamServerInfo{
				FullMethod:     method,
				IsClientStream: registered.description.ClientStreams,
				IsServerStream: registered.description.ServerStreams,
			}
			err = interceptor(registered.service, stream, info, func(service any, serverStream grpc.ServerStream) error {
				return registered.description.Handler(service, serverStream)
			})
		}()
		stream.finish(err)
	}()
	return clientStream, nil
}

type localStream struct {
	ctx             context.Context
	description     *grpc.StreamDesc
	clientMessages  chan proto.Message
	serverMessages  chan proto.Message
	clientSendDone  chan struct{}
	done            chan struct{}
	headerReady     chan struct{}
	mu              sync.Mutex
	headerOnce      sync.Once
	finishedErr     error
	finished        bool
	clientSendClose bool
	header          metadata.MD
	trailer         metadata.MD
	headerSent      bool
	operation       string
}

type localClientStream struct {
	*localStream
	headerOptions  []*metadata.MD
	trailerOptions []*metadata.MD
}

// SetContext 更新进程内 stream 的服务端上下文。
func (s *localStream) SetContext(ctx context.Context) {
	s.mu.Lock()
	s.ctx = ctx
	s.mu.Unlock()
}

// SetHeader 累积服务端流响应头。
func (s *localStream) SetHeader(md metadata.MD) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finished || s.headerSent {
		return status.Error(codes.Internal, "进程内 gRPC 流式响应头已发送")
	}
	s.header = metadata.Join(s.header, md)
	return nil
}

// SendHeader 发送服务端流响应头。
func (s *localStream) SendHeader(md metadata.MD) error {
	s.mu.Lock()
	if s.finished || s.headerSent {
		s.mu.Unlock()
		return status.Error(codes.Internal, "进程内 gRPC 流式响应头重复发送")
	}
	s.header = metadata.Join(s.header, md)
	s.headerSent = true
	s.mu.Unlock()
	s.markHeaderReady()
	return nil
}

// SetTrailer 累积服务端流尾部元数据。
func (s *localStream) SetTrailer(md metadata.MD) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.finished {
		s.trailer = metadata.Join(s.trailer, md)
	}
}

// Context 返回本地流上下文。
func (s *localStream) Context() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

// Method 返回本地 stream 的完整 gRPC 方法名。
func (s *localStream) Method() string { return s.operation }

// SendMsg 发送服务端流消息。
func (s *localStream) SendMsg(message any) error {
	protobufMessage, ok := message.(proto.Message)
	if !ok {
		return fmt.Errorf("进程内 gRPC 仅支持 protobuf 消息，来源 %T", message)
	}
	s.mu.Lock()
	if s.finished {
		err := s.finishedErr
		s.mu.Unlock()
		if err != nil {
			return err
		}
		return io.EOF
	}
	s.headerSent = true
	s.mu.Unlock()
	s.markHeaderReady()
	select {
	case s.serverMessages <- proto.Clone(protobufMessage):
		return nil
	case <-s.Context().Done():
		return s.Context().Err()
	case <-s.done:
		return streamResult(s.result())
	}
}

// RecvMsg 接收客户端发送给服务端的流消息。
func (s *localStream) RecvMsg(message any) error {
	received, err := s.receiveClientMessage()
	if err != nil {
		return err
	}
	return copyMessage(message, received)
}

// SendMsg 发送客户端流消息。
func (s *localClientStream) SendMsg(message any) error {
	protobufMessage, ok := message.(proto.Message)
	if !ok {
		return fmt.Errorf("进程内 gRPC 仅支持 protobuf 消息，来源 %T", message)
	}
	s.mu.Lock()
	closed := s.clientSendClose
	s.mu.Unlock()
	if closed {
		return io.EOF
	}
	select {
	case s.clientMessages <- proto.Clone(protobufMessage):
		return nil
	case <-s.Context().Done():
		return s.Context().Err()
	case <-s.done:
		return streamResult(s.result())
	}
}

// RecvMsg 接收服务端发送给客户端的流消息。
func (s *localClientStream) RecvMsg(message any) error {
	received, err := s.receiveServerMessage()
	if err != nil {
		s.updateTrailerOptions()
		return err
	}
	s.updateTrailerOptions()
	return copyMessage(message, received)
}

// CloseSend 关闭客户端发送方向。
func (s *localClientStream) CloseSend() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clientSendClose {
		return nil
	}
	s.clientSendClose = true
	close(s.clientSendDone)
	return nil
}

// Header 返回服务端流响应头。
func (s *localClientStream) Header() (metadata.MD, error) {
	select {
	case <-s.headerReady:
	case <-s.done:
	case <-s.Context().Done():
		return nil, s.Context().Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	header := metadata.Join(s.header)
	for _, target := range s.headerOptions {
		*target = metadata.Join(header)
	}
	return header, nil
}

// Trailer 返回服务端流尾部元数据。
func (s *localClientStream) Trailer() metadata.MD {
	s.mu.Lock()
	defer s.mu.Unlock()
	trailer := metadata.Join(s.trailer)
	for _, target := range s.trailerOptions {
		*target = metadata.Join(trailer)
	}
	return trailer
}

// updateTrailerOptions 将当前 stream 尾部元数据回填到 CallOption 指针。
func (s *localClientStream) updateTrailerOptions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	trailer := metadata.Join(s.trailer)
	for _, target := range s.trailerOptions {
		*target = metadata.Join(trailer)
	}
}

// configureCallOptions 保存本地 stream 调用需要回填的响应元数据目标。
func (s *localClientStream) configureCallOptions(options []grpc.CallOption) {
	for _, option := range options {
		switch value := option.(type) {
		case grpc.HeaderCallOption:
			if value.HeaderAddr != nil {
				s.headerOptions = append(s.headerOptions, value.HeaderAddr)
			}
		case grpc.TrailerCallOption:
			if value.TrailerAddr != nil {
				s.trailerOptions = append(s.trailerOptions, value.TrailerAddr)
			}
		case *grpc.HeaderCallOption:
			if value != nil && value.HeaderAddr != nil {
				s.headerOptions = append(s.headerOptions, value.HeaderAddr)
			}
		case *grpc.TrailerCallOption:
			if value != nil && value.TrailerAddr != nil {
				s.trailerOptions = append(s.trailerOptions, value.TrailerAddr)
			}
		}
	}
}

func (s *localStream) finish(err error) {
	s.mu.Lock()
	s.finishedErr = err
	s.finished = true
	s.mu.Unlock()
	s.markHeaderReady()
	close(s.done)
}

func (s *localStream) result() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finishedErr
}

func (s *localStream) markHeaderReady() {
	s.headerOnce.Do(func() { close(s.headerReady) })
}

// applyCallOptions 将本地 unary 响应元数据回填到 gRPC CallOption 指针。
func applyCallOptions(options []grpc.CallOption, response *ResponseMetadata) {
	if response == nil {
		return
	}
	for _, option := range options {
		switch value := option.(type) {
		case grpc.HeaderCallOption:
			if value.HeaderAddr != nil {
				*value.HeaderAddr = metadata.Join(response.Header)
			}
		case grpc.TrailerCallOption:
			if value.TrailerAddr != nil {
				*value.TrailerAddr = metadata.Join(response.Trailer)
			}
		case *grpc.HeaderCallOption:
			if value != nil && value.HeaderAddr != nil {
				*value.HeaderAddr = metadata.Join(response.Header)
			}
		case *grpc.TrailerCallOption:
			if value != nil && value.TrailerAddr != nil {
				*value.TrailerAddr = metadata.Join(response.Trailer)
			}
		}
	}
}

func (s *localStream) receiveClientMessage() (proto.Message, error) {
	var err error
	select {
	case message := <-s.clientMessages:
		return message, nil
	default:
	}
	select {
	case message := <-s.clientMessages:
		return message, nil
	case <-s.clientSendDone:
		select {
		case message := <-s.clientMessages:
			return message, nil
		default:
			return nil, io.EOF
		}
	case <-s.Context().Done():
		return nil, s.Context().Err()
	case <-s.done:
		err = s.result()
		if err != nil {
			return nil, err
		}
		select {
		case message := <-s.clientMessages:
			return message, nil
		default:
			return nil, io.EOF
		}
	}
}

func (s *localStream) receiveServerMessage() (proto.Message, error) {
	var err error
	select {
	case message := <-s.serverMessages:
		return message, nil
	default:
	}
	select {
	case message := <-s.serverMessages:
		return message, nil
	case <-s.Context().Done():
		return nil, s.Context().Err()
	case <-s.done:
		err = s.result()
		if err != nil {
			return nil, err
		}
		select {
		case message := <-s.serverMessages:
			return message, nil
		default:
			return nil, io.EOF
		}
	}
}

func streamResult(err error) error {
	if err != nil {
		return err
	}
	return io.EOF
}

func copyMessage(target, source any) error {
	targetMessage, targetOK := target.(proto.Message)
	sourceMessage, sourceOK := source.(proto.Message)
	if !targetOK || !sourceOK {
		return fmt.Errorf("进程内 gRPC 仅支持 protobuf 消息，来源 %T，目标 %T", source, target)
	}
	if targetMessage.ProtoReflect().Descriptor().FullName() != sourceMessage.ProtoReflect().Descriptor().FullName() {
		return fmt.Errorf("进程内 gRPC 消息类型不一致，来源 %s，目标 %s", sourceMessage.ProtoReflect().Descriptor().FullName(), targetMessage.ProtoReflect().Descriptor().FullName())
	}
	proto.Reset(targetMessage)
	proto.Merge(targetMessage, sourceMessage)
	return nil
}

func newLocalStream(ctx context.Context, description *grpc.StreamDesc, operation string) *localStream {
	return &localStream{
		ctx: ctx, description: description, operation: operation, clientMessages: make(chan proto.Message, 1), serverMessages: make(chan proto.Message, 1),
		clientSendDone: make(chan struct{}), done: make(chan struct{}), headerReady: make(chan struct{}),
	}
}

func (c *Conn) registerUnary(serviceName string, method *grpc.MethodDesc, service any) {
	fullMethod := "/" + serviceName + "/" + method.MethodName
	if _, exists := c.methods[fullMethod]; exists {
		panic(fmt.Sprintf("localgrpc: gRPC 方法 %s 重复注册", fullMethod))
	}
	if _, exists := c.streams[fullMethod]; exists {
		panic(fmt.Sprintf("localgrpc: gRPC 方法 %s 重复注册", fullMethod))
	}
	c.methods[fullMethod] = unaryMethod{description: method, service: service}
}
