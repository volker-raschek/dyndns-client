package updater

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"git.cryptic.systems/volker.raschek/dyndns-client/pkg/types"
)

type Updater interface {
	AddRecord(ctx context.Context, name string, ttl uint, recordType string, data string) error
	DeleteRecord(ctx context.Context, name string, recordType string) error
	PruneRecords(ctx context.Context, name string) error
}

type NSUpdate struct {
	server  string
	tsigKey *types.TSIGKey
}

func (u *NSUpdate) AddRecord(ctx context.Context, name string, ttl uint, recordType string, data string) error {
	nsUpdateCmd := ""
	switch recordType {
	case "A", "AAAA":
		nsUpdateCmd = fmt.Sprintf("update add %v %v IN %v %v", name, ttl, recordType, data)
	case "TXT":
		nsUpdateCmd = fmt.Sprintf("update add %v %v IN %v \"%v\"", name, ttl, recordType, data)
	default:
		return fmt.Errorf("RecordType %v not supported", recordType)
	}

	return u.execute(ctx, nsUpdateCmd)
}

func (u *NSUpdate) DeleteRecord(ctx context.Context, name string, recordType string) error {
	nsUpdateCmd := fmt.Sprintf("update delete %v IN %v", name, recordType)
	return u.execute(ctx, nsUpdateCmd)
}

func (u *NSUpdate) execute(ctx context.Context, nsUpdateCmd string) error {
	body := fmt.Sprintf("server %v\n%v\nsend\nquit", u.server, nsUpdateCmd)

	errBuffer := new(bytes.Buffer)

	// #nosec G204
	cmd := exec.CommandContext(ctx, "nsupdate", "-y", fmt.Sprintf("%v:%v:%v", u.tsigKey.Algorithm, u.tsigKey.Name, u.tsigKey.Secret))
	// cmd.Stdout = os.Stdout
	cmd.Stderr = bufio.NewWriter(errBuffer)
	cmd.Stdin = strings.NewReader(body)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("nsupdate error %w: %v", err, errBuffer.String())
	}

	return nil
}

func (u *NSUpdate) PruneRecords(ctx context.Context, name string) error {
	nsUpdateCmd := fmt.Sprintf("update delete %v", name)
	return u.execute(ctx, nsUpdateCmd)
}

func NewNSUpdate(server string, tsigKey *types.TSIGKey) (Updater, error) {
	return &NSUpdate{
		server:  server,
		tsigKey: tsigKey,
	}, nil
}
