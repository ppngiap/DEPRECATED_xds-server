package apiv1

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iotbzh/xds-server/lib/common"
)

// ExecArgs JSON parameters of /exec command
type ExecArgs struct {
	ID         string   `json:"id"`
	RPath      string   `json:"rpath"` // relative path into project
	Cmd        string   `json:"cmd" binding:"required"`
	Args       []string `json:"args"`
	CmdTimeout int      `json:"timeout"` // command completion timeout in Second
}

// ExecOutMsg Message send on each output (stdout+stderr) of executed command
type ExecOutMsg struct {
	CmdID     string `json:"cmdID"`
	Timestamp string `json:"timestamp"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
}

// ExecExitMsg Message send when executed command exited
type ExecExitMsg struct {
	CmdID     string `json:"cmdID"`
	Timestamp string `json:"timestamp"`
	Code      int    `json:"code"`
	Error     error  `json:"error"`
}

// ExecOutEvent Event send in WS when characters are received
const ExecOutEvent = "exec:output"

// ExecExitEvent Event send in WS when program exited
const ExecExitEvent = "exec:exit"

var execCommandID = 1

// ExecCmd executes remotely a command
func (s *APIService) execCmd(c *gin.Context) {
	var args ExecArgs
	if c.BindJSON(&args) != nil {
		common.APIError(c, "Invalid arguments")
		return
	}

	// TODO: add permission

	// Retrieve session info
	sess := s.sessions.Get(c)
	if sess == nil {
		common.APIError(c, "Unknown sessions")
		return
	}
	sop := sess.IOSocket
	if sop == nil {
		common.APIError(c, "Websocket not established")
		return
	}

	// Allow to pass id in url (/exec/:id) or as JSON argument
	id := c.Param("id")
	if id == "" {
		id = args.ID
	}
	if id == "" {
		common.APIError(c, "Invalid id")
		return
	}

	prj := s.cfg.GetFolderFromID(id)
	if prj == nil {
		common.APIError(c, "Unknown id")
		return
	}

	execTmo := args.CmdTimeout
	if execTmo == 0 {
		// TODO get default timeout from config.json file
		execTmo = 24 * 60 * 60 // 1 day
	}

	// Define callback for output
	var oCB common.EmitOutputCB
	oCB = func(sid string, id int, stdout, stderr string) {
		// IO socket can be nil when disconnected
		so := s.sessions.IOSocketGet(sid)
		if so == nil {
			s.log.Infof("%s not emitted: WS closed - sid: %s - msg id:%d", ExecOutEvent, sid, id)
			return
		}
		s.log.Debugf("%s emitted - WS sid %s - id:%d", ExecOutEvent, sid, id)

		// FIXME replace by .BroadcastTo a room
		err := (*so).Emit(ExecOutEvent, ExecOutMsg{
			CmdID:     strconv.Itoa(id),
			Timestamp: time.Now().String(),
			Stdout:    stdout,
			Stderr:    stderr,
		})
		if err != nil {
			s.log.Errorf("WS Emit : %v", err)
		}
	}

	// Define callback for output
	eCB := func(sid string, id int, code int, err error) {
		s.log.Debugf("Command [Cmd ID %d] exited: code %d, error: %v", id, code, err)

		// IO socket can be nil when disconnected
		so := s.sessions.IOSocketGet(sid)
		if so == nil {
			s.log.Infof("%s not emitted - WS closed (id:%d", ExecExitEvent, id)
			return
		}

		// FIXME replace by .BroadcastTo a room
		e := (*so).Emit(ExecExitEvent, ExecExitMsg{
			CmdID:     strconv.Itoa(id),
			Timestamp: time.Now().String(),
			Code:      code,
			Error:     err,
		})
		if e != nil {
			s.log.Errorf("WS Emit : %v", e)
		}
	}

	cmdID := execCommandID
	execCommandID++

	cmd := "cd " + prj.GetFullPath(args.RPath) + " && " + args.Cmd
	if len(args.Args) > 0 {
		cmd += " " + strings.Join(args.Args, " ")
	}

	s.log.Debugf("Execute [Cmd ID %d]: %v %v", cmdID, cmd)
	err := common.ExecPipeWs(cmd, sop, sess.ID, cmdID, execTmo, s.log, oCB, eCB)
	if err != nil {
		common.APIError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK,
		gin.H{
			"status": "OK",
			"cmdID":  cmdID,
		})
}
