package instance_service

import (
	"errors"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	whatsmeow_service "github.com/evolution-foundation/evolution-go/pkg/whatsmeow/service"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
)

type fakeWhatsmeow struct {
	whatsmeow_service.WhatsmeowService
	reconnects   int
	starts       int
	reconnectErr error
}

func (f *fakeWhatsmeow) ReconnectClient(string) error {
	f.reconnects++
	return f.reconnectErr
}

func (f *fakeWhatsmeow) StartInstance(string) error {
	f.starts++
	return nil
}

func newReconnectService(t *testing.T, client *whatsmeow.Client) (instances, *fakeWhatsmeow) {
	t.Helper()
	fake := &fakeWhatsmeow{}
	pointers := map[string]*whatsmeow.Client{}
	if client != nil {
		pointers["inst"] = client
	}
	return instances{
		clientPointer:    pointers,
		whatsmeowService: fake,
		loggerWrapper:    logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: t.TempDir()}),
	}, fake
}

// Queda de 04/11: o religamento automático falhou, o cliente ficou na memória desconectado
// e /instance/reconnect respondia "client disconnected" sem nunca religar.
func TestReconnectRestartsDisconnectedClientInMemory(t *testing.T) {
	svc, fake := newReconnectService(t, whatsmeow.NewClient(&store.Device{}, nil))

	if err := svc.Reconnect(&instance_model.Instance{Id: "inst"}); err != nil {
		t.Fatalf("esperava sucesso, veio: %v", err)
	}
	if fake.reconnects != 1 {
		t.Fatalf("ReconnectClient chamado %d vezes, esperava 1", fake.reconnects)
	}
	if fake.starts != 0 {
		t.Fatalf("StartInstance chamado %d vezes, esperava 0", fake.starts)
	}
}

func TestReconnectReturnsRestartError(t *testing.T) {
	svc, fake := newReconnectService(t, whatsmeow.NewClient(&store.Device{}, nil))
	fake.reconnectErr = errors.New("falhou")

	if err := svc.Reconnect(&instance_model.Instance{Id: "inst"}); !errors.Is(err, fake.reconnectErr) {
		t.Fatalf("esperava o erro do ReconnectClient, veio: %v", err)
	}
}

// Caminho usado em 03/11, sem cliente na memória: continua igual.
func TestReconnectWithoutClientStillStartsInstance(t *testing.T) {
	svc, fake := newReconnectService(t, nil)

	if err := svc.Reconnect(&instance_model.Instance{Id: "inst"}); err == nil {
		t.Fatal("o cliente falso não sobe: esperava erro de validação")
	}
	if fake.starts != 1 {
		t.Fatalf("StartInstance chamado %d vezes, esperava 1", fake.starts)
	}
	if fake.reconnects != 0 {
		t.Fatalf("ReconnectClient chamado %d vezes, esperava 0", fake.reconnects)
	}
}
