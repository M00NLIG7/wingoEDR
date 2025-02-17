package modes

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"wingoEDR/backup"
	"wingoEDR/chainsaw"
	"wingoEDR/common"
	"wingoEDR/processes"
	"wingoEDR/usermanagement"

	"github.com/olekukonko/tablewriter"
	"go.uber.org/zap"
)

func ModeHandler(mode string, otherParams map[string]string) {

	switch mode {
	case "backup":
		zap.S().Infof("Mode is %s", mode)
		BackupMode(otherParams)
	case "chainsaw":
		zap.S().Infof("Mode is %s", mode)
		Chainsaw(otherParams)
	case "sessions":
		zap.S().Infof("Mode is %s", mode)
		SessionsMode()
	case "userenum":
		zap.S().Infof("Mode is %s", mode)
		UserEnumMode()
	case "processexplorer":
		zap.S().Infof("Mode is %s", mode)
		ProcessExplorerMode()
	case "decompress":
		zap.S().Infof("Mode is %s", mode)
		DecompressMode(otherParams)
	default:
		zap.S().Infof("No mode selected defaulting to continious monitoring")
		return

	}
	os.Exit(0)
}

func BackupMode(otherParams map[string]string) {
	common.VerifyWindowsPathFatal(otherParams["backupDir"])
	common.VerifyWindowsPathFatal(otherParams["backupItem"])

	file, err := os.Open(otherParams["backupItem"])
	if err != nil {
		zap.S().Fatal("Backup item file access failure! Err: %v", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		zap.S().Fatal("Backup item file access failure! Err: %v", err)
	}

	if fileInfo.IsDir() { // Direcotry backups not quite working consult Ethan
		backup.BackDir(otherParams["backupItem"], false)
		zap.S().Infof("Backup of %s is complete!", otherParams["backupItem"])
	} else {
		newFileName := "\\compressed_" + fileInfo.Name()
		backup.BackFile(newFileName, otherParams["backupItem"])
		zap.S().Infof("[INFO]	Backup of %s is complete!", otherParams["backupItem"])
	}

	os.Exit(0)
}

func Chainsaw(otherParams map[string]string) {
	// Required params check
	if otherParams["from"] != "" {
		if otherParams["to"] != "" {
			chainsaw.ScanTimeRange(otherParams["from"], otherParams["to"])
		} else {
			zap.S().Fatal("Missing required param: to")
		}

	} else {
		events, err := chainsaw.ScanAll()
		if err != nil {
			zap.S().Fatal("Chainsaw events were not scanned: ", err.Error())
		}

		table := tablewriter.NewWriter(os.Stdout)

		table.SetHeader([]string{"Timestamp", "RuleName", "Tags", "Authors"})

		for _, e := range events {
			row := []string{e.Timestamp, e.RuleName, strings.Join(e.Tags, ","), strings.Join(e.Authors, ",")}
			table.Append(row)
		}

		table.SetRowLine(true)

		table.SetRowSeparator("-")
		table.Render()

	}

	os.Exit(0)
}

func SessionsMode() {
	sessions := usermanagement.ListSessions()

	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Username", "Domain", "LocalUser", "LocalAdmin", "LogonType", "LogonTime", "DnsDomainName"})

	for _, s := range sessions {
		row := []string{s.Username, s.Domain, strconv.FormatBool(s.LocalUser), strconv.FormatBool(s.LocalAdmin), strconv.FormatUint(uint64(s.LogonType), 10), s.LogonTime.String(), s.DnsDomainName}
		table.Append(row)
	}

	table.Render()
	os.Exit(0)
}

func UserEnumMode() {
	users := usermanagement.ReturnUsers()

	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Username", "Fullname", "Enabled", "Locked", "Admin", "Passwdexpired", "CantChangePasswd", "Passwdage", "Lastlogon", "BadPasswdAttemps", "NumofLogons"})

	for _, u := range users {
		row := []string{u.Username, u.Fullname, strconv.FormatBool(u.Enabled), strconv.FormatBool(u.Locked), strconv.FormatBool(u.Admin), strconv.FormatBool(u.Passwdexpired), strconv.FormatBool(u.CantChangePasswd), u.Passwdage.String(), u.Lastlogon.String(), strconv.FormatUint(uint64(u.BadPasswdAttempts), 10), strconv.FormatUint(uint64(u.NumofLogons), 10)}
		table.Append(row)
	}

	table.Render()
	os.Exit(0)
}

func DecompressMode(otherParams map[string]string) {
	// Required params check
	common.VerifyWindowsPathFatal(otherParams["decompressitem"])

	reader, err := os.Open(otherParams["decompressitem"])
	if err != nil {
		zap.S().Fatal("Backup item file access failure! Err: %v", err)
	}

	file := filepath.Base(otherParams["decompressitem"])
	newFileName := file[11:]
	writer, err := os.Create(newFileName)
	if err != nil {
		zap.S().Fatal("Backup item file access failure! Err: %v", err)
	}
	common.Decompress(reader, writer)
	os.Exit(0)
}

func ProcessExplorerMode() {
	processes, err := processes.GetAllProcesses()
	if err != nil {
		zap.S().Error("WingoEDR has encountered and error: ", err)
	}

	for _, processInfo := range processes {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "PPID", "PID", "Executable", "Working Directory", "User"})

		var ppid string
		if processInfo.Ppid != nil {
			ppid = fmt.Sprintf("%d", processInfo.Ppid.Pid)
		} else {
			ppid = "N/A"
		}

		exe := strings.Join(processInfo.Exe, " ")

		table.Append([]string{
			processInfo.Name,
			ppid,
			fmt.Sprintf("%d", processInfo.Pid),
			exe,
			processInfo.Cwd,
			processInfo.User,
		})

		if len(processInfo.NetworkConnections) > 0 {
			table.Append([]string{"", "", "", "", "", ""})
			table.Append([]string{"Network Connections", "", "", "", "", ""})
			table.SetHeader([]string{"Name", "PPID", "PID", "Executable", "Working Directory", "User", "Net Type", "Local Address", "Local Port", "Remote Address", "Remote Port", "Status"})

			for _, conn := range processInfo.NetworkConnections {
				table.Append([]string{
					processInfo.Name,
					ppid,
					fmt.Sprintf("%d", processInfo.Pid),
					exe,
					processInfo.Cwd,
					processInfo.User,
					fmt.Sprintf("%d", conn.NetType),
					conn.LocalAddress,
					fmt.Sprintf("%d", conn.LocalPort),
					conn.RemoteAddress,
					fmt.Sprintf("%d", conn.RemotePort),
					conn.Status,
				})
			}
		}

		table.Render()
	}

	os.Exit(0)
}
