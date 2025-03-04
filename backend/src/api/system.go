package api

import (
	"bufio"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/go-fuego/fuego"
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
	fuego.PutStd(srv, "/restart", p.RestartHandler)
	fuego.GetStd(srv, "/nics", p.GetNICsHandler)
	fuego.GetStd(srv, "/filesystems", p.GetFSHandler)
	return nil
}

// RestartHandler godoc
//
//	@Summary		RestartHandler
//	@Description	Restart the server ( useful in development )
//	@Tags			system
//	@Produce		json
//	@Success		204
//	@Failure		405	{object}	dto.ErrorInfo
//	@Router			/restart [put]
func (handler *SystemHanler) RestartHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)

	slog.Debug("Restarting server...")
	overseer.Restart()
}

// GetNICsHandler godoc
//
//	@Summary		GetNICsHandler
//	@Description	Return all network interfaces
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	dto.NetworkInfo
//	@Failure		405	{object}	dto.ErrorInfo
//	@Router			/nics [get]
func (handler *SystemHanler) GetNICsHandler(w http.ResponseWriter, r *http.Request) {

	net, err := ghw.Network()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	var info dto.NetworkInfo
	var conv converter.NetToDtoImpl
	err = conv.NetInfoToNetworkInfo(*net, &info)
	//err = mapper.Map(context.Background(), &info, net)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, info, nil)
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

// GetFSHandler godoc
//
//	@Summary		GetFSHandler
//	@Description	Return all supported fs
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	dto.FilesystemTypes
//	@Failure		405	{object}	dto.ErrorInfo
//	@Router			/filesystems [get]
func (handler *SystemHanler) GetFSHandler(w http.ResponseWriter, r *http.Request) {

	fs, err := handler.getFileSystems()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var xfs dto.FilesystemTypes
	for _, fsi := range fs {
		xfs = append(xfs, dto.FilesystemType(fsi))
	}
	//err = mapper.Map(context.Background(), &xfs, fs)
	//if err != nil {
	//	HttpJSONReponse(w, err, nil)
	//	return
	//}
	HttpJSONReponse(w, xfs, nil)
}
