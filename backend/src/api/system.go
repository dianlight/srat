package api

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/jaypipes/ghw"
	"github.com/jpillora/overseer"
)

type SystemHanler struct {
}

func NewSystemHanler() *SystemHanler {
	p := new(SystemHanler)

	return p
}

func (p *SystemHanler) Routers(srv *fuego.Server) error {
	fuego.Put(srv, "/restart", p.RestartHandler, option.Description("Restart the server ( useful in development )"), option.Tags("dev"))
	fuego.Get(srv, "/nics", p.GetNICsHandler, option.Description("Return all network interfaces"), option.Tags("system"))
	fuego.Get(srv, "/filesystems", p.GetFSHandler, option.Description("Return all supported fs"), option.Tags("system"))
	return nil
}

func (handler *SystemHanler) RestartHandler(c fuego.ContextNoBody) (bool, error) {
	slog.Debug("Restarting server...")
	overseer.Restart()
	return true, nil
}

func (handler *SystemHanler) GetNICsHandler(c fuego.ContextNoBody) (*dto.NetworkInfo, error) {

	net, err := ghw.Network()
	if err != nil {
		return nil, err
	}

	var info dto.NetworkInfo
	var conv converter.NetToDtoImpl
	err = conv.NetInfoToNetworkInfo(*net, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// ReadLinesOffsetN reads contents from file and splits them by new line.
// The offset tells at which line number to start.
// The count determines the number of lines to read (starting from offset):
// n >= 0: at most n lines
// n < 0: whole file
// Source: https://github.com/shirou/gopsutil
func (handler *SystemHanler) readLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for i := uint(0); i < uint(n)+offset || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				ret = append(ret, strings.Trim(line, "\n"))
			}
			break
		}
		if i < offset {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}

// Source: https://github.com/shirou/gopsutil
func (handler *SystemHanler) getFileSystems() ([]string, error) {
	filename := "/proc/filesystems"
	lines, err := handler.readLinesOffsetN(filename, 0, -1)
	if err != nil {
		return nil, err
	}
	var ret []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "nodev") {
			ret = append(ret, strings.TrimSpace(line))
			continue
		}
		t := strings.Split(line, "\t")
		if len(t) != 2 || t[1] != "zfs" {
			continue
		}
		ret = append(ret, strings.TrimSpace(t[1]))
	}

	return ret, nil
}

func (handler *SystemHanler) GetFSHandler(c fuego.ContextNoBody) (dto.FilesystemTypes, error) {

	fs, err := handler.getFileSystems()
	if err != nil {
		return nil, err
	}
	var xfs dto.FilesystemTypes
	for _, fsi := range fs {
		xfs = append(xfs, dto.FilesystemType(fsi))
	}
	return xfs, nil
}
