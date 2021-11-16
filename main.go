package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"github.com/lxn/win"

	gcfg "gopkg.in/gcfg.v1"
)

import (
	"bufio"
	"encoding/json"

	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"./telnet"
	"./utils"
)

var (
	gLogger                      *log.Logger
	gLogFile                     *os.File
	configFile                   *Config
	mw                           *MyMainWindow
	RootCmdPre                   string
	chiptype                     string
	localIp                      string
	localPath                    string
	mutableCondition             *walk.MutableCondition
	telnetEdit, pcTextEdit       *walk.TextEdit
	macLineEdit                  *walk.LineEdit
	label5gUpRlt, label5gDownRlt *walk.Label
	c                            *telnet.Client
	db                           *walk.DataBinder
	OTTIperfCmd                  string
	ottIperfUpCmd24              string
	ottIperfUpCmd5               string
	CatBobSnCmd                  string
)

var burnsnflag = false
var amlcatbobsnflag = false
var PING_DISMISS_DLG bool = false
var DEBUG bool = false

func initLog() {
	var logFilename = "debug.log"
	if _, err := os.Stat(logFilename); err == nil {
		fmt.Println("debug file exists")
		os.Remove(logFilename)
	}
	gLogFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	// defer gLogFile.Close()

	if err != nil {
		log.Fatal("open log file error: %v", err)
		os.Exit(-1)
	}

	writers := []io.Writer{
		gLogFile,
		os.Stdout,
	}
	fileAndStdoutWriter := io.MultiWriter(writers...)
	log.SetPrefix("[Debug]")
	gLogger = log.New(fileAndStdoutWriter, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

type IperfConfig struct {
	OttIp         string
	OttPort       string
	OttInterval   string
	OttRuntime    string
	OttLimitspeed string
	runSync       string
}

type Config struct {
	PcConfig struct {
		LocalPort       string
		CheckWifiStatic bool
		AgingTime       string
		NormalDelay     string
		LongDelay       string
		Passcount       string
		Dispasscount    string
		Passrate        string
		ReportData      bool
		PingNetCount    string
		RetestTimes     string
		ButtonText      string
		ButtonIcon      string
		PingNetTimes    string
		TipRetest       string
		RetestItem      bool
	}
	Wifi24Config struct {
		Wifi24GIp           string
		Ssid24              string
		LimitSpeed24        string
		LimitSpeed24Up      string
		LimitSpeed24Down    string
		LimitSpeed24UpRlt   string
		LimitSpeed24DownRlt string
		Button24UpIcon      string
		Button24DownIcon    string
	}
	Wifi5Config struct {
		Wifi5GIp           string
		Ssid5              string
		LimitSpeed5        string
		LimitSpeed5Up      string
		LimitSpeed5Down    string
		LimitSpeed5UpRlt   string
		LimitSpeed5DownRlt string
		Button5UpIcon      string
		Button5DownIcon    string
	}
	PublicConfig struct {
		OttEthIp      string
		Ott24Port     string
		Ott5Port      string
		RunMinutes    string
		RunSeconds    string
		Interval      string
		FrequencyType string
		PackageMethod string
		WifiChip      string
		SyncRun       bool
		Sn            string
		Bobsn         string
	}
}

type OttpackageBean struct {
	BatchNo               string `json:"batchNo"`
	Bobsn                 string `json:"bobsn"`
	Boxno                 int    `json:"boxno"`
	Chflag                string `json:"chflag"`
	Contractordernumber   string `json:"contractordernumber"`
	Createdatetime        string `json:"createdatetime"`
	Createuser            string `json:"createuser"`
	Mac                   string `json:"mac"`
	Prefecturelevelcityid string `json:"prefecturelevelcityid"`
	ProvinceId            string `json:"provinceId"`
	Sn                    string `json:"sn"`
	Stbid                 string `json:"stbid"`
	Tysn                  string `json:"tysn"`
}

type RequestBean struct {
	Dut_Sn  		string `json:"Dut_Sn"`
	Pre_Station		string `json:"Pre_Station"`
	Cur_Station   	string `json:"Cur_Station"`
	Status  		string `json:"Status"`
}

// responseBean
type ResponseBean struct {
	IsSuccess       bool			`json:"IsSuccess"`
	Message    		string         	`json:"Message"`
	//Ottpackage OttpackageBean `json:ottpackage`
}

// test iperf result Bean
type IperfResult struct {
	upRlt    string
	upUnit   string
	downRlt  string
	downUnit string
}

func initIni() {
	configFile = &Config{}
	configFileName := "config.ini"
	if _, err := os.Stat(configFileName); err == nil {
		err := gcfg.ReadFileInto(configFile, configFileName)
		if err != nil {
			gLogger.Fatal("Failed to parse config file: %s", err)
		}
		gLogger.Println("configFile is ", configFile)
	} else {
		// configFile, err := os.OpenFile(configFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		// gcfg.set()
	}

}

func initWindows() {
	macLineEdit.SetFocus()
	focused := macLineEdit.Focused()
	gLogger.Println("focused:", focused)
}

func getLocalIp() (IpAddr string) {
	addrSlice, err := net.InterfaceAddrs()
	if nil != err {
		gLogger.Println("Get local IP addr failed!!!")
		IpAddr = "localhost"
		return IpAddr
	}
	IpAddr = ""
	for _, addr := range addrSlice {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if nil != ipnet.IP.To4() {
				if strings.Contains(ipnet.IP.String(), "192.168.2") {
					IpAddr += ipnet.IP.String()
				}
			}
		}
	}
	return IpAddr
}

func main() {
	initLog()
	initIni()
	OTTIperfCmd = "iperf -s -p" + configFile.PublicConfig.Ott24Port + " -m -u"
	localIp = getLocalIp()
	localPath = GetCurPath()
	gLogger.Println("local path is:", localPath)
	mw = new(MyMainWindow)
	mutableCondition = walk.NewMutableCondition()
	if !DEBUG {
		mutableCondition.SetSatisfied(true)
	}
	MustRegisterCondition("allWidgetReadonly", mutableCondition)
	var openAction *walk.Action

	var logoFile = localPath + "/img/logo.ico"
	if _, err := os.Stat(logoFile); err != nil {
		gLogger.Println("logo:", err.Error())
		walk.MsgBox(mw, "提示", "img文件夹被移除", walk.MsgBoxIconWarning)
		logoFile = ""
		// return
	} else {
		logoFile = "/img/logo.ico"
	}

	if _, err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "亨谷吞吐量测试工具",
		Background: SolidColorBrush{
			Color: walk.RGB(255, 255, 255),
		},
		Icon: logoFile,
		Functions: map[string]func(args ...interface{}) (interface{}, error){
			"initFocus": func(args ...interface{}) (interface{}, error) {
				initWindows()
				gLogger.Println("get here:", args[0])

				return "stop", nil
			},
		},
		Persistent:         true,
		RightToLeftReading: false,
		MenuItems: []MenuItem{
			Menu{
				Text: "&文件",
				Items: []MenuItem{
					Action{
						AssignTo:    &openAction,
						Text:        "&吞吐量设置",
						Image:       "img/open.png",
						OnTriggered: mw.openPass_Triggered,
					},
					Separator{},
					Action{
						Text:        "退出",
						OnTriggered: func() { mw.Close() },
					},
				},
			},
			Menu{
				Text: "&帮助",
				Items: []MenuItem{
					Action{
						Text:        "关于",
						OnTriggered: mw.aboutAction_Triggered,
					},
				},
			},
		},

		// MinSize: Size{1000, 500},
		Size: Size{1350, 550},
		DataBinder: DataBinder{
			AssignTo:       &db,
			Name:           "configFile",
			DataSource:     configFile,
			AutoSubmit:     true,
			ErrorPresenter: ToolTipErrorPresenter{},
			OnDataSourceChanged: func() {
				gLogger.Println("OnDataSourceChanged: ", configFile)
			},
			OnSubmitted: func() {
				gLogger.Println("OnSubmitted new config is: ", configFile)
			},
			OnCanSubmitChanged: func() {
				gLogger.Println("OnCanSubmitChanged new config is: ", configFile)
			},
		},
		Layout: VBox{Alignment: AlignHFarVFar},
		Children: []Widget{
			Composite{
				Layout: HBox{Alignment: AlignHNearVCenter, Spacing: 5},
				Children: []Widget{

					Composite{
						Border: true,
						Layout: Grid{
							Alignment: AlignHNearVNear,
							Columns:   2},
						Children: []Widget{
							LineEdit{
								AssignTo:   &macLineEdit,
								ColumnSpan: 2,
								CueBanner:  "请输入bobsn码:",
								CaseMode:   CaseModeUpper,
								MaxSize:    Size{200, 20},
								MinSize:    Size{100, 10},
								OnKeyDown: func(key walk.Key) {
									if key == walk.KeyReturn {
										gLogger.Println("enter is press")
										macstr := macLineEdit.Text()
										gLogger.Println("len is:", strings.Count(macstr, ""), "macstr is:", macstr)
										if macstr != "" &&
											((strings.Count(macstr, "") == 13 &&
												!strings.Contains(macstr, ":")) ||
												(strings.Count(macstr, "") == 18 &&
													strings.Contains(macstr, ":"))) {
											gLogger.Println("mac is :" + macstr)
											if strings.Contains(macstr, ":") {
												macstr = strings.ReplaceAll(macstr, ":", "")
											}
											gLogger.Println("new mac is :" + macstr)
											body := mw.httpDo("GET", "http://192.168.2.101:9080/ottpack/OTT/GetDataByMac/"+macstr, "")
											gLogger.Println("response:", body)

											/*if body != "" {
												var response ResponseBean
												json.Unmarshal([]byte(body), &response)
												gLogger.Println("response parse is:", response)
												if response.Code == 0 {
													if response.Ottpackage.Sn == "" || response.Ottpackage.Bobsn == "" {
														walk.MsgBox(mw, "sn未设置，请检查", response.Message, walk.MsgBoxIconWarning)
													} else {
														configFile.PublicConfig.Sn = response.Ottpackage.Sn
														configFile.PublicConfig.Bobsn = response.Ottpackage.Bobsn
														db.Reset()
														clearRlt()
														go mw.startIperf(response.Ottpackage.Sn, response.Ottpackage.Bobsn)
													}

												} else if response.Code == -1 {
													configFile.PublicConfig.Sn = ""
													configFile.PublicConfig.Bobsn = ""
													db.Reset()
													walk.MsgBox(mw, "信息未入库", response.Message, walk.MsgBoxIconWarning)
												}

											}*/

										} else if macstr != "" &&
											strings.Count(macstr, "") == 15 {
											// write bobsn from scan gun
											if configFile.PcConfig.ReportData {
												request := RequestBean{
													Dut_Sn: macstr,
													Pre_Station:  "",
													Cur_Station:  "WIFI_THRO",
												}
												jsonRequest, err := json.Marshal(request)
												if err != nil {
													gLogger.Println("生成json字符串错误")
												}
												body := mw.httpDo("POST", "http://192.168.2.101:8888/DBATE/CheckStation", string(jsonRequest))
												gLogger.Println("response:", body)

												if body != "" {
													var response ResponseBean
													json.Unmarshal([]byte(body), &response)
													gLogger.Println("CheckStation parse is:", response)
													if response.IsSuccess {
														configFile.PublicConfig.Bobsn = macstr
														db.Reset()
														clearRlt()
														go mw.startIperf("", macstr)
													} else {
														go walk.MsgBox(mw, "检查站点", "服务器提示:"+response.Message, walk.MsgBoxIconWarning)
													}
												}
											} else {
												configFile.PublicConfig.Bobsn = macstr
												db.Reset()
												clearRlt()
												go mw.startIperf("", macstr)
											}
										} else {
											macLineEdit.SetTextSelection(0, strings.Count(macstr, ""))
										}
									}
								},
							},

							Label{
								ColumnSpan: 2,
								Text:       "即将写入",
							},
							Label{
								ColumnSpan: 1,
								Text:       "sn:",
							},
							Label{
								ColumnSpan: 1,
								Text:       Bind("configFile.PublicConfig.Sn"),
							},
							Label{
								ColumnSpan: 1,
								Text:       "bobsn:",
							},
							Label{
								ColumnSpan: 1,
								Text:       Bind("configFile.PublicConfig.Bobsn"),
							},
						},
					},

					Composite{
						Border: true,
						Layout: VBox{Spacing: 5, Alignment: AlignHNearVNear},
						Children: []Widget{
							GroupBox{
								Title:  "2.4Gwifi設置",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{
										Text: "SSID",
									},
									LineEdit{
										MaxSize:    Size{100, 20},
										CueBanner:  "scty-2.4",
										Text:       Bind("configFile.Wifi24Config.Ssid24"),
										Persistent: false,
										ReadOnly:   Bind("allWidgetReadonly"),
									},
									Label{
										Text: "Ip",
									},
									LineEdit{
										Text:     Bind("configFile.Wifi24Config.Wifi24GIp"),
										MaxSize:  Size{100, 20},
										ReadOnly: Bind("allWidgetReadonly"),
									},
								},
							},

							GroupBox{
								Title:  "5Gwifi配置",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{
										Text: "SSID",
									},
									LineEdit{
										Text:       Bind("configFile.Wifi5Config.Ssid5"),
										MaxSize:    Size{100, 20},
										CueBanner:  "scty-2.4",
										Persistent: false,
										ReadOnly:   Bind("allWidgetReadonly"),
									},
									Label{
										Text: "Ip",
									},
									LineEdit{
										Text:     Bind("configFile.Wifi5Config.Wifi5GIp"),
										MaxSize:  Size{100, 20},
										ReadOnly: Bind("allWidgetReadonly"),
									},
								},
							},
						},
					},
					Composite{
						Border: true,
						Layout: VBox{Spacing: 5, Alignment: AlignHNearVNear},
						Children: []Widget{
							GroupBox{
								Title:  "本机配置",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{
										Text: "IP",
									},
									LineEdit{
										Text:       localIp,
										MaxSize:    Size{100, 20},
										CueBanner:  "192.168.1.3",
										Persistent: false,
										ReadOnly:   Bind("allWidgetReadonly"),
									},
									Label{
										Text: "端口",
									},
									LineEdit{
										Text:     Bind("configFile.PcConfig.LocalPort"),
										MaxSize:  Size{100, 20},
										ReadOnly: Bind("allWidgetReadonly"),
									},
								},
							},
							GroupBox{
								Title:  "机顶盒配置",
								Layout: Grid{Columns: 4},
								Children: []Widget{
									Label{
										ColumnSpan: 1,
										Text:       "IP",
									},
									LineEdit{
										ColumnSpan: 3,
										Text:       Bind("configFile.PublicConfig.OttEthIp"),
										MaxSize:    Size{100, 20},
										CueBanner:  "192.168.1.2",
										Persistent: false,
										ReadOnly:   Bind("allWidgetReadonly"),
									},
									Label{
										ColumnSpan: 1,
										Text:       "2.4g端口",
									},
									LineEdit{
										ColumnSpan: 1,
										Text:       Bind("configFile.PublicConfig.Ott24Port"),
										MaxSize:    Size{100, 20},
										ReadOnly:   Bind("allWidgetReadonly"),
									},
									Label{
										ColumnSpan: 1,
										Text:       "5g端口",
									},
									LineEdit{
										ColumnSpan: 1,
										Text:       Bind("configFile.PublicConfig.Ott5Port"),
										MaxSize:    Size{100, 20},
										ReadOnly:   Bind("allWidgetReadonly"),
									},
								},
							},
						},
					},
					GroupBox{
						Title:  "时间配置",
						Layout: Grid{Columns: 5},
						Children: []Widget{
							Label{
								Text: "时长：",
							},
							LineEdit{
								Text:       Bind("configFile.PublicConfig.RunMinutes"),
								MaxSize:    Size{20, 20},
								CueBanner:  "0",
								Persistent: false,
								ReadOnly:   Bind("allWidgetReadonly"),
							},
							Label{
								Text: "分",
							},
							LineEdit{
								Text:       Bind("configFile.PublicConfig.RunSeconds"),
								MaxSize:    Size{20, 20},
								CueBanner:  "10",
								Persistent: false,
								ReadOnly:   Bind("allWidgetReadonly"),
							},
							Label{
								Text: "秒",
							},
							Label{
								ColumnSpan: 3,
								Text:       "统计间隔：",
							},
							LineEdit{
								Text:     Bind("configFile.PublicConfig.Interval"),
								MaxSize:  Size{20, 20},
								ReadOnly: Bind("allWidgetReadonly"),
							},
							Label{
								Text: "秒",
							},
							Label{
								ColumnSpan: 3,
								Text:       "老化卡值:",
							},
							LineEdit{
								Text:      Bind("configFile.PcConfig.AgingTime"),
								CueBanner: "14400",
								MaxSize:   Size{50, 20},
								ReadOnly:  Bind("allWidgetReadonly"),
							},
							Label{
								Text: "秒",
							},
							CheckBox{
								ColumnSpan: 5,
								Text:       "无线静态卡站",
								Checked:    Bind("configFile.PcConfig.CheckWifiStatic"),
								Enabled:    Bind("!allWidgetReadonly"),
							},
							Label{
								ColumnSpan: 3,
								Text:       "连接wifi重复测试:",
							},
							LineEdit{
								Text:      Bind("configFile.PcConfig.RetestTimes"),
								CueBanner: "5",
								MaxSize:   Size{50, 20},
								ReadOnly:  Bind("allWidgetReadonly"),
							},
							Label{
								Text: "次",
							},
						},
					},
					GroupBox{
						Title:  "频段配置",
						Layout: Grid{Columns: 6, Alignment: AlignHNearVCenter},
						Children: []Widget{
							Label{
								ColumnSpan: 1,
								Text:       "方式",
							},
							ComboBox{
								Name:        "cbbf",
								ColumnSpan:  5,
								Value:       Bind("configFile.PublicConfig.FrequencyType"),
								ToolTipText: "方式",
								Model:       []string{"单频", "双频"},
								Enabled:     Bind("!allWidgetReadonly"),
								OnCurrentIndexChanged: func() {
								},
							},
							CheckBox{
								Name:               "cbd",
								TextOnLeftSide:     false,
								RightToLeftReading: true,
								ColumnSpan:         3,
								Text:               "上下行同时进行",
								Checked:            Bind("configFile.PublicConfig.SyncRun"),
								OnCheckStateChanged: func() {
									if mutableCondition.Satisfied() {
										return
									}
								},
								Enabled: Bind("!allWidgetReadonly"),
							},
							ComboBox{
								Name:       "cbpack",
								ColumnSpan: 3,
								Value:      Bind("configFile.PublicConfig.PackageMethod"),

								ToolTipText: "发包方式",
								Model:       []string{"tcp", "udp"},
								Enabled:     Bind("!allWidgetReadonly"),
								OnCurrentIndexChanged: func() {
								},
							},
							Label{
								ColumnSpan: 1,
								Text:       "2.4G指定宽带速率:",
							},

							LineEdit{
								ColumnSpan: 1,
								Text:       Bind("configFile.Wifi24Config.LimitSpeed24"),
								CueBanner:  "80",
								MaxSize:    Size{50, 20},
								ReadOnly:   Bind("allWidgetReadonly"),
							},
							Label{
								ColumnSpan: 1,
								Text:       "M",
							},
							Label{
								ColumnSpan: 1,
								Text:       "5G指定宽带速率:",
								Visible:    Bind("cbbf.Value=='双频'"),
							},

							LineEdit{
								ColumnSpan: 1,
								Text:       Bind("configFile.Wifi5Config.LimitSpeed5"),
								CueBanner:  "400",
								MaxSize:    Size{50, 20},
								Visible:    Bind("cbbf.Value=='双频'"),
								ReadOnly:   Bind("allWidgetReadonly"),
							},
							Label{
								ColumnSpan: 1,
								Text:       "M",
								Visible:    Bind("cbbf.Value=='双频'"),
							},

							Label{
								ColumnSpan: 1,
								Text:       "2.4G上行速率对比值:",
							},

							LineEdit{
								ColumnSpan: 1,
								Text:       Bind("configFile.Wifi24Config.LimitSpeed24Up"),
								CueBanner:  "40",
								MaxSize:    Size{50, 20},
								ReadOnly:   Bind("allWidgetReadonly"),
							},
							Label{
								ColumnSpan: 1,
								Text:       "M",
							},
							Label{
								ColumnSpan: 1,
								Text:       "5G上行速率对比值:",
								Visible:    Bind("cbbf.Value=='双频'"),
							},

							LineEdit{
								ColumnSpan: 1,
								Text:       Bind("configFile.Wifi5Config.LimitSpeed5Up"),
								CueBanner:  "100",
								MaxSize:    Size{50, 20},
								Visible:    Bind("cbbf.Value=='双频'"),
								ReadOnly:   Bind("allWidgetReadonly"),
							},
							Label{
								ColumnSpan: 1,
								Visible:    Bind("cbbf.Value=='双频'"),
								Text:       "M",
							},

							Label{
								ColumnSpan: 1,
								Text:       "2.4G下行速率对比值:",
							},

							LineEdit{
								ColumnSpan: 1,
								Text:       Bind("configFile.Wifi24Config.LimitSpeed24Down"),
								CueBanner:  "20",
								MaxSize:    Size{50, 20},
								ReadOnly:   Bind("allWidgetReadonly"),
							},
							Label{
								ColumnSpan: 1,
								Text:       "M",
							},
							Label{
								ColumnSpan: 1,
								Text:       "5G下行速率对比值:",
								Visible:    Bind("cbbf.Value=='双频'"),
							},

							LineEdit{
								ColumnSpan: 1,
								Text:       Bind("configFile.Wifi5Config.LimitSpeed5Down"),
								CueBanner:  "50",
								MaxSize:    Size{50, 20},
								Visible:    Bind("cbbf.Value=='双频'"),
								ReadOnly:   Bind("allWidgetReadonly"),
							},
							Label{
								ColumnSpan: 1,
								Text:       "M",
								Visible:    Bind("cbbf.Value=='双频'"),
							},
						},
					},
					Composite{
						Layout: Grid{Columns: 3, Alignment: AlignHNearVCenter},
						Children: []Widget{

							PushButton{
								ColumnSpan: 3,
								Image:      Bind("configFile.PcConfig.ButtonIcon"),
								Text:       Bind("configFile.PcConfig.ButtonText"),
								MaxSize:    Size{100, 50},
								OnClicked: func() {
									if configFile.PcConfig.ButtonText == "开始" {
										configFile.PcConfig.ButtonText = "暂停"
										configFile.PcConfig.ButtonIcon = "img/button_pause_32.png"
										if !mutableCondition.Satisfied() {
											mutableCondition.SetSatisfied(true)
										}
										go mw.startIperf("", "")
									} else {
										configFile.PcConfig.ButtonText = "开始"
										configFile.PcConfig.ButtonIcon = "img/button_play_32px.png"
									}
									clearRlt()
								},
							},
						},
					},
				},
			},
			Composite{
				Layout:  HBox{Alignment: AlignHNearVNear, Spacing: 15},
				MinSize: Size{1350, 400},
				MaxSize: Size{1350, 450},
				Children: []Widget{
					TextEdit{
						// StretchFactor: 1,
						MaxSize:  Size{450, 450},
						AssignTo: &telnetEdit,
						VScroll:  true,
						OnTextChanged: func() {
						},
					},

					// VSpacer{Size: 2},
					TextEdit{
						MaxSize:  Size{450, 450},
						AssignTo: &pcTextEdit,
						// StretchFactor: 1,
						VScroll: true,
						Text:    "",
					},

					Composite{
						// StretchFactor: 1,
						Border:  true,
						MaxSize: Size{450, 450},
						Layout:  VBox{Spacing: 5, Alignment: AlignHNearVNear},
						Children: []Widget{
							GroupBox{
								Title:   "2.4G测试结果:",
								MaxSize: Size{450, 150},
								Layout:  Grid{Columns: 3},
								Children: []Widget{
									Label{Text: "2.4G上行:", ColumnSpan: 1},
									Label{Text: Bind("configFile.Wifi24Config.LimitSpeed24UpRlt"), ColumnSpan: 1,
										TextAlignment: AlignFar},
									ImageView{Image: Bind("configFile.Wifi24Config.Button24UpIcon"), ColumnSpan: 1,
										Mode: ImageViewModeShrink},
									Label{Text: "2.4G下行:", ColumnSpan: 1},
									Label{Text: Bind("configFile.Wifi24Config.LimitSpeed24DownRlt"), ColumnSpan: 1,
										TextAlignment: AlignFar},
									ImageView{Image: Bind("configFile.Wifi24Config.Button24DownIcon"), ColumnSpan: 1,
										Mode: ImageViewModeShrink},
								}},
							GroupBox{
								Title:   "5G测试结果:",
								MaxSize: Size{450, 150},
								Layout:  Grid{Columns: 3},
								Children: []Widget{
									Label{Text: "5G上行:", ColumnSpan: 1},
									Label{AssignTo: &label5gUpRlt, Text: Bind("configFile.Wifi5Config.LimitSpeed5UpRlt"), ColumnSpan: 1,
										TextAlignment: AlignFar},
									ImageView{Image: Bind("configFile.Wifi5Config.Button5UpIcon"), ColumnSpan: 1,
										Mode: ImageViewModeShrink},
									Label{Text: "5G下行:", ColumnSpan: 1},
									Label{AssignTo: &label5gDownRlt, Text: Bind("configFile.Wifi5Config.LimitSpeed5DownRlt"), ColumnSpan: 1,
										TextAlignment: AlignFar},
									ImageView{Image: Bind("configFile.Wifi5Config.Button5DownIcon"), ColumnSpan: 1,
										Mode: ImageViewModeShrink},
								}},
							GroupBox{
								Title:   "统计",
								MaxSize: Size{450, 150},
								Layout:  Grid{Columns: 3},
								Children: []Widget{
									Label{Text: "合格:", ColumnSpan: 1},
									HSpacer{Size: 60, ColumnSpan: 1},
									Label{Text: Bind("configFile.PcConfig.Passcount"), ColumnSpan: 1, TextAlignment: AlignFar},
									Label{Text: "不合格:", ColumnSpan: 1},
									HSpacer{Size: 60, ColumnSpan: 1},
									Label{Text: Bind("configFile.PcConfig.Dispasscount"), ColumnSpan: 1, TextAlignment: AlignFar},
									Label{Text: "合格率:", ColumnSpan: 1},
									HSpacer{Size: 60, ColumnSpan: 1},
									Label{Text: Bind("configFile.PcConfig.Passrate"), ColumnSpan: 1, TextAlignment: AlignFar},
								},
							},
						},
					},
				},
			},
		},
	}.Run()); err != nil {
		gLogger.Fatal(err)
	}
}

type MyMainWindow struct {
	*walk.MainWindow
	prevFilePath string
}

func (mw *MyMainWindow) openPass_Triggered() {
	var dlg *walk.Dialog
	var ltpwd *walk.LineEdit

	if _, err := (Dialog{
		AssignTo: &dlg,
		Title:    "请输入密码",
		Icon:     "/img/logo.ico",
		MinSize:  Size{200, 150},
		Size:     Size{200, 150},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: "请输入密码：",
			},
			LineEdit{
				// Name:       "lepwd",
				AssignTo:   &ltpwd,
				MaxSize:    Size{100, 20},
				Persistent: false,
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						Text: "确定",
						OnClicked: func() {
							timeStr := time.Now().Format("020106")
							timeInt, errint := strconv.Atoi(timeStr)
							if errint != nil {
								gLogger.Fatal(errint)
								walk.MsgBox(mw, "提示", "解析本机时间故障", walk.MsgBoxIconQuestion)
								dlg.Cancel()
							}

							timesqrt := math.Sqrt(float64(timeInt))
							timesqrtInt := int(timesqrt)
							pwdF := timesqrt - float64(timesqrtInt)
							pwdStr := strconv.FormatFloat(pwdF, 'f', 100, 64)
							gLogger.Println("pwdStr:", pwdStr)
							pwdLong := strings.Replace(pwdStr, "0.", "", -1)
							pwd := pwdLong[:6]
							gLogger.Println("pwd", pwd)
							if ltpwd.Text() == pwd {
								mutableCondition.SetSatisfied(false)
								dlg.Cancel()
							} else {
								gLogger.Println("no!!")
								walk.MsgBox(mw, "提示", "密码错误，请重新输入(⊙o⊙)", walk.MsgBoxIconQuestion|walk.MsgBoxOK)
							}
						},
					},
					PushButton{
						Text: "取消",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}.Run(mw)); err != nil {
		gLogger.Fatal(err)
	}
}

func (mw *MyMainWindow) aboutAction_Triggered() {
	walk.MsgBox(mw, "关于", "版本号v2.0-2020-7-23_0922\n张聪聪版权所有@2020\n"+
		"技术支持QQ:2353410167", walk.MsgBoxIconInformation)
}

const (
	GetChiptype     = "getprop ro.product.chiptype"
	CheckIperTest   = "cat /data/local/iperf-test"
	CheckAgingCmd   = "cat /data/local/aging-test"
	CheckWifiStatic = "ll /data/local/wifi_scanid"
	ConnectWifiCmd  = "facwifitools  ssid "
	StopCmd         = "\x03"
)

func dealWithCmdStr(msg, cmd string) string {
	if strings.Contains(msg, "") {
		msg = strings.Replace(msg, "\r", "", -1)
		msg = strings.Replace(msg, "\n", "", -1)
		re := regexp.MustCompile("\\s+<+")
		msg = re.ReplaceAllString(msg, "")
		gLogger.Println("replace reg:", msg)

		cmdarr := strings.Split(cmd, "&&")
		gLogger.Println("cmdarr:", cmdarr)
		regCmdStr := cmdarr[0] + ".*" + cmdarr[len(cmdarr) - 1]
		gLogger.Println("regCmdStr:", regCmdStr)
		regCmd := regexp.MustCompile(regCmdStr)
		msg = regCmd.ReplaceAllString(msg, "")

		gLogger.Println("<<111:", msg)
		msg = strings.Replace(msg, RootCmdPre, "", 2)
		msg = strings.Replace(msg, " ", "", -1)
		gLogger.Println("<<222:", msg)
		return msg
	}
	rltIndexPre := strings.Index(msg, RootCmdPre+cmd)
	rltIndexLast := strings.LastIndex(msg, RootCmdPre)
	preLength := strings.Count(RootCmdPre+cmd, "")

	newMsg := msg[rltIndexPre+preLength : rltIndexLast]
	/*gLogger.Println("rltIndexPre:", rltIndexPre)
	gLogger.Println("preLength:", preLength)
	gLogger.Println("rltIndexLast:", rltIndexLast)*/

	newMsg = strings.Replace(newMsg, "\r", "", -1)
	newMsg = strings.Replace(newMsg, "\n", "", -1)
	newMsg = strings.Replace(newMsg, "", "", -1)
	gLogger.Println("newMsg:", newMsg)

	return newMsg
}

func UserOutput(recv []byte) {

TRY:

	begin, end := -1, -1
	for i, v := range recv {

		switch v {
		case 27:
			begin = i
		case 'm':
			end = i
		case 7:
			recv[i] = ' '
		}

		if begin != -1 && end != -1 && begin < end {

			if begin == 0 {
				recv = recv[end+1:]
			} else if end+1 >= len(recv) {
				recv = recv[0:begin]
			} else {
				recv = append(recv[0:begin], recv[end+1:]...)
			}

			goto TRY
		}
	}

	if len(recv) > 3 {

		//gLogger.Println("recv str:", len(recv), string(recv))
		// gLogger.Println("edit :", telnetEdit.TextLength())
		recvStr := string(recv)
		telnetEdit.AppendText(recvStr)
		s, e := telnetEdit.TextSelection()
		telnetEdit.SetTextSelection(s, e)

		telnetEditStr := telnetEdit.Text()
		gLogger.Println("telnetEditStr str:", telnetEditStr)

		if RootCmdPre == "" {
			if strings.Contains(telnetEditStr, "root") {
				RootCmdPre = telnetEditStr
				RootCmdPre = strings.Replace(RootCmdPre, "\r", "", -1)
				RootCmdPre = strings.Replace(RootCmdPre, "\n", "", -1)
				gLogger.Println("RootCmdPre:", RootCmdPre)
				telnetEdit.SetText("")
				UserInput("\n" + GetChiptype + "\r")
			}
		}

		if RootCmdPre != "" && strings.Count(telnetEditStr, RootCmdPre) == 2 {
			if strings.Contains(telnetEditStr, GetChiptype) {
				//0, get chiptype & rootcmd
				chiptype = dealWithCmdStr(telnetEditStr, CheckIperTest)
				telnetEdit.SetText("")
				if burnsnflag {
					var inputSnCmd, inputBobsnCmd string
					inputSnCmd = ""
					inputBobsnCmd = ""
					if strings.Contains(chiptype, "Amlogic") {
						if configFile.PublicConfig.Sn != "" {
							inputSnCmd = "echo 1 > /sys/class/unifykeys/attach&&" +
								"echo \"usid\" > /sys/class/unifykeys/name&&echo " +
								configFile.PublicConfig.Sn + " > /sys/class/unifykeys/write"
						}
						if configFile.PublicConfig.Bobsn != "" {
							inputBobsnCmd = "echo 1 > /sys/class/unifykeys/attach&&" +
								"echo \"tystbid\" > /sys/class/unifykeys/name&&echo " +
								configFile.PublicConfig.Bobsn + " > /sys/class/unifykeys/write"
						}

					} else if strings.Contains(chiptype, "Hi3798") {
						if configFile.PublicConfig.Sn != "" {
							inputSnCmd = "mtdinfo set sn " + configFile.PublicConfig.Sn
						}
						if configFile.PublicConfig.Bobsn != "" {
							inputBobsnCmd = "mtdinfo set tyid " + configFile.PublicConfig.Bobsn
						}
					}
					if inputSnCmd != "" {
						UserInput(inputSnCmd + "\r")
					}
					if inputBobsnCmd != "" {
						UserInput("\n" + inputBobsnCmd + "\r")
					}
				} else {
					if strings.Contains(chiptype, "Amlogic") {
						amlcatbobsnflag = true
						CatBobSnCmd =  "echo 1 > /sys/class/unifykeys/attach&&" +
							"echo tystbid > /sys/class/unifykeys/name&&" +
							"cat /sys/class/unifykeys/read"

					} else if strings.Contains(chiptype, "Hi3798") {
						CatBobSnCmd = "mtdinfo get tyid"
					}
					if CatBobSnCmd != "" {
						UserInput("\n" + CatBobSnCmd + "\r")
					}
				}
			} else if strings.Contains(telnetEditStr, CatBobSnCmd) || amlcatbobsnflag {
				amlcatbobsnflag = false
				telnetEdit.SetText("")
				bobsn := dealWithCmdStr(telnetEditStr, CatBobSnCmd)
				gLogger.Println("bobsn:", bobsn)
				if bobsn != "" && strings.Contains(bobsn, "tyid:") {
					bobsn = strings.Replace(bobsn, "tyid:", "", -1)
					bobsn = strings.Replace(bobsn, " ", "", -1)
					gLogger.Println("hisi new bobsn:", bobsn)
				}
				// write bobsn from system
				if configFile.PcConfig.ReportData {
					request := RequestBean{
						Dut_Sn: bobsn,
						Pre_Station:  "",
						Cur_Station:  "WIFI_THRO",
					}
					jsonRequest, err := json.Marshal(request)
					if err != nil {
						gLogger.Println("生成json字符串错误")
					}
					body := mw.httpDo("POST", "http://192.168.2.101:8888/DBATE/CheckStation", string(jsonRequest))
					gLogger.Println("response:", body)

					if body != "" {
						var response ResponseBean
						json.Unmarshal([]byte(body), &response)
						gLogger.Println("CheckStation parse is:", response)
						if response.IsSuccess {
							configFile.PublicConfig.Bobsn = bobsn
							db.Reset()
							UserInput("\n" + CheckIperTest + "\r")
						} else {
							go walk.MsgBox(mw, "检查站点", "服务器提示:"+response.Message, walk.MsgBoxIconWarning)
							return
						}
					}
				} else {
					configFile.PublicConfig.Bobsn = bobsn
					db.Reset()
					UserInput("\n" + CheckIperTest + "\r")
				}
			} else if burnsnflag && strings.Contains(telnetEditStr, configFile.PublicConfig.Bobsn) {
				telnetEdit.SetText("")
				UserInput("\n" + CheckIperTest + "\r")
			} else if strings.Contains(telnetEditStr, CheckIperTest) {
				//①first step, check iperf result
				iperfRlt := dealWithCmdStr(telnetEditStr, CheckIperTest)
				gLogger.Println("check iperf result:", iperfRlt)
				if iperfRlt == "1" && configFile.PcConfig.TipRetest == "true" {
					reTestDiaInt := walk.MsgBox(mw, "提示", "已测试，是否重测", walk.MsgBoxIconWarning|walk.MsgBoxOKCancel)
					gLogger.Println("dialog int:", reTestDiaInt)
					if reTestDiaInt == 1 {
						telnetEdit.SetText("")
						UserInput("\n" + CheckAgingCmd + "\r")
					} else if reTestDiaInt == 2 {
						return
					}
				} else {
					telnetEdit.SetText("")
					UserInput("\n" + CheckAgingCmd + "\r")
				}
			} else if strings.Contains(telnetEditStr, CheckAgingCmd) {
				//② second step, check aging time
				agingRlt := dealWithCmdStr(telnetEditStr, CheckAgingCmd)
				// agingRltTrim := strings.Replace(agingRlt, " ", "", -1)
				// gLogger.Println("agingRlt:", agingRltTrim)
				agingTime, errint := strconv.Atoi(agingRlt)
				if errint != nil {
					walk.MsgBox(mw, "提示", "老化时间解析失败", walk.MsgBoxIconWarning)
					return
				}
				if agingTime < 14400 && configFile.PcConfig.AgingTime != "-1"{
					walk.MsgBox(mw, "提示", "老化时间不足请检查！", walk.MsgBoxIconWarning)
					return
				} else {
					gLogger.Println("checkwifi static:", configFile.PcConfig.CheckWifiStatic)
					if configFile.PcConfig.CheckWifiStatic {
						telnetEdit.SetText("")
						UserInput("\n" + CheckWifiStatic + "\r")
					} else {
						//④ fourth step,connect wifi
						telnetEdit.SetText("")
						if configFile.PublicConfig.WifiChip == "realtek" {
							UserInput("\n" + "echo 0 >/proc/net/rtl88x2cs/wlan0/scan_deny&&\n" +
								ConnectWifiCmd + configFile.Wifi24Config.Ssid24 + "\r")
						} else {
							UserInput("\n" + ConnectWifiCmd + configFile.Wifi24Config.Ssid24 + "\r")
						}

					}
				}
			} else if strings.Contains(telnetEditStr, CheckWifiStatic) {
				//③ third step, check wifi static
				wifiStaticRlt := dealWithCmdStr(telnetEditStr, CheckWifiStatic)
				if strings.Contains(wifiStaticRlt, "No such file or directory") {
					walk.MsgBox(mw, "提示", "无线静态未测试请检查！", walk.MsgBoxIconWarning)
					return
				} else {
					//④ fourth step,connect wifi
					telnetEdit.SetText("")
					if configFile.PublicConfig.WifiChip == "realtek" {
						UserInput("\n" + "echo 0 >/proc/net/rtl88x2cs/wlan0/scan_deny&&\n" +
							ConnectWifiCmd + configFile.Wifi24Config.Ssid24 + "\r")
					} else {
						UserInput("\n" + ConnectWifiCmd + configFile.Wifi24Config.Ssid24 + "\r")
					}
				}
			} else if strings.Contains(telnetEditStr, ConnectWifiCmd) {

				if strings.Contains(telnetEditStr, configFile.Wifi24Config.Ssid24) {
					//⑤ fifth step, test 2.4G run iperf
					wifiConnectIp := dealWithCmdStr(telnetEditStr, ConnectWifiCmd+configFile.Wifi24Config.Ssid24)
					gLogger.Println("wifiConnectIp:", wifiConnectIp)
					if configFile.PublicConfig.WifiChip == "realtek" {
						wifiIndex := strings.Index(wifiConnectIp, configFile.Wifi24Config.Ssid24)
						wifiConnectIp = wifiConnectIp[wifiIndex+len(configFile.Wifi24Config.Ssid24):]
						gLogger.Println("deal wifiConnectIp:", wifiConnectIp)
					}
					configFile.Wifi24Config.Wifi24GIp = wifiConnectIp

					err := db.Reset()
					if err != nil {
						gLogger.Fatal("wifi connect err:", err)
					}
					if strings.Contains(wifiConnectIp, "255|") {
						walk.MsgBox(mw, "提示", "wifi连接失败请重试！", walk.MsgBoxIconWarning)
						return
					}
					telnetEdit.SetText("")
					if configFile.PublicConfig.SyncRun {
						if configFile.PublicConfig.PackageMethod == "tcp" {
							var pcIperfUpCmd = "cmd /c " + localPath + "/iperf.exe -c " +
								configFile.Wifi24Config.Wifi24GIp + " -i " +
								configFile.PublicConfig.Interval + " -t " +
								configFile.PublicConfig.RunSeconds + " -p" +
								configFile.PublicConfig.Ott24Port + " -L" +
								configFile.PcConfig.LocalPort + " -d -f m"
							if configFile.PublicConfig.WifiChip == "realtek" {
								ottIperfUpCmd24 = "echo 1 > /proc/net/rtl88x2cs/wlan0/scan_deny&&" +
									"echo 1 > /proc/net/rtl88x2cs/wlan0/tx_quick_addba_req&&" +
									"cat /proc/net/rtl88x2cs/wlan0/scan_abort&&" +
									"\niperf -s -p" + configFile.PublicConfig.Ott24Port
							} else {
								ottIperfUpCmd24 = "iperf -s -p" + configFile.PublicConfig.Ott24Port
							}

							checkIperfCmd := exec.Command("cmd", "/c", "TASKLIST", "|findstr", "iperf.exe", ">a.txt")
							checkIperfCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
							checkIperfCmd.Run()
							aFileName := "a.txt"
							if _, err := os.Stat(aFileName); err == nil {
								aFileSize := getSize(localPath + "/" + aFileName)
								aFileContent, _ := ReadAll(localPath + "/" + aFileName)
								if aFileSize != 0 && strings.Contains(string(aFileContent), "iperf.exe") {
									killiperfcmd := exec.Command("cmd", "/c", "TASKKILL", "/IM", "iperf.exe", "/F")
									killiperfcmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
									err := killiperfcmd.Run()
									if err == nil {
										gLogger.Println("kill iperf\r")
										UserInput("\n" + ottIperfUpCmd24 + "\r")
										ottIperfUpCmd24 = "iperf -s -p" + configFile.PublicConfig.Ott24Port
										time.Sleep(time.Duration(2) * time.Second)
										go runIperfCmdInPc(pcIperfUpCmd)
										gLogger.Println("2.4G pcIperfcmd:", ottIperfUpCmd24)
									}

								} else {
									UserInput("\n" + ottIperfUpCmd24 + "\r")
									ottIperfUpCmd24 = "iperf -s -p" + configFile.PublicConfig.Ott24Port
									time.Sleep(time.Duration(2) * time.Second)
									go runIperfCmdInPc(pcIperfUpCmd)
									gLogger.Println("2.4G pcIperfcmd:", ottIperfUpCmd24)
								}

							} else {
								UserInput("\n" + ottIperfUpCmd24 + "\r")
								ottIperfUpCmd24 = "iperf -s -p" + configFile.PublicConfig.Ott24Port
								time.Sleep(time.Duration(2) * time.Second)
								go runIperfCmdInPc(pcIperfUpCmd)
								gLogger.Println("2.4G pcIperfcmd:", ottIperfUpCmd24)
							}

						} else {
							//udp
							var pcIperfUpCmd = "cmd /c " + localPath + "/iperf.exe -s -p" +
								configFile.PcConfig.LocalPort + " -m -u"
							ottIperfUpCmd24 = "iperf -c " + localIp + " -p" +
								configFile.PcConfig.LocalPort + " -i " +
								configFile.PublicConfig.Interval + " -t " +
								configFile.PublicConfig.RunSeconds + " -u -b " +
								configFile.Wifi24Config.LimitSpeed24 + "M -d -L " +
								configFile.PublicConfig.Ott24Port

							checkIperfCmd := exec.Command("cmd", "/c", "TASKLIST", "|findstr", "iperf.exe", ">a.txt")
							checkIperfCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
							checkIperfCmd.Run()
							aFileName := "a.txt"
							if _, err := os.Stat(aFileName); err == nil {
								aFileSize := getSize(localPath + "/" + aFileName)
								aFileContent, _ := ReadAll(localPath + "/" + aFileName)
								if aFileSize != 0 && strings.Contains(string(aFileContent), "iperf.exe") {
									killiperfcmd := exec.Command("cmd", "/c", "TASKKILL", "/IM", "iperf.exe", "/F")
									killiperfcmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
									err := killiperfcmd.Run()
									if err == nil {
										gLogger.Println("kill iperf\r")
										go runIperfCmdInPc(pcIperfUpCmd)
										time.Sleep(time.Duration(2) * time.Second)
										UserInput("\n" + ottIperfUpCmd24 + "\r")
										gLogger.Println("2.4G pcIperfcmd:", ottIperfUpCmd24)
									}

								} else {
									go runIperfCmdInPc(pcIperfUpCmd)
									time.Sleep(time.Duration(2) * time.Second)
									UserInput("\n" + ottIperfUpCmd24 + "\r")
									gLogger.Println("2.4G pcIperfcmd:", ottIperfUpCmd24)
								}

							} else {
								go runIperfCmdInPc(pcIperfUpCmd)
								time.Sleep(time.Duration(2) * time.Second)
								UserInput("\n" + ottIperfUpCmd24 + "\r")
								gLogger.Println("2.4G pcIperfcmd:", ottIperfUpCmd24)
							}
							ottIperfUpCmd24 = "iperf -c " + localIp + " -p" +
								configFile.PcConfig.LocalPort + " -i " +
								configFile.PublicConfig.Interval + " -t " +
								configFile.PublicConfig.RunSeconds + " -u -b " +
								configFile.Wifi24Config.LimitSpeed24 + "M"
						}

					} else {
						//test up first
						var pcIperfUpCmd = "cmd /c " + localPath + "/iperf.exe -s -p" +
							configFile.PublicConfig.Ott24Port + " -m -u"
						go runIperfCmdInPc(pcIperfUpCmd)
						ottIperfUpCmd24 = "iperf -c " + localIp + " -p" +
							configFile.PublicConfig.Ott24Port + " -i " +
							configFile.PublicConfig.Interval + " -t " +
							configFile.PublicConfig.RunSeconds + " -u -b " +
							configFile.Wifi24Config.LimitSpeed24 + "M -L " +
							configFile.PcConfig.LocalPort
						gLogger.Println("2.4G upcmd in ott:", ottIperfUpCmd24)
						UserInput("\n" + ottIperfUpCmd24 + "\r")
					}

				} else if strings.Contains(telnetEditStr, configFile.Wifi5Config.Ssid5) {
					//⑥ sixth step, test 5G
					wifiConnectIp := dealWithCmdStr(telnetEditStr, ConnectWifiCmd+configFile.Wifi5Config.Ssid5)
					gLogger.Println("5g wifiConnectIp:", wifiConnectIp)
					if configFile.PublicConfig.WifiChip == "realtek" {
						wifiIndex := strings.Index(wifiConnectIp, configFile.Wifi5Config.Ssid5)
						wifiConnectIp = wifiConnectIp[wifiIndex+len(configFile.Wifi5Config.Ssid5):]
						gLogger.Println("deal 5g wifiConnectIp:", wifiConnectIp)
					}
					configFile.Wifi5Config.Wifi5GIp = wifiConnectIp

					err := db.Reset()
					if err != nil {
						gLogger.Fatal("wifi connect err:", err)
					}
					if strings.Contains(wifiConnectIp, "255|") {
						walk.MsgBox(mw, "提示", "wifi连接失败请重试！", walk.MsgBoxIconWarning)
						return
					}
					telnetEdit.SetText("")

					if configFile.PublicConfig.PackageMethod == "tcp" {
						var pcIperfUpCmd = "cmd /c " + localPath + "/iperf.exe -c " +
							configFile.Wifi5Config.Wifi5GIp + " -i " +
							configFile.PublicConfig.Interval + " -t " +
							configFile.PublicConfig.RunSeconds + " -p" +
							configFile.PublicConfig.Ott5Port + " -L" +
							configFile.PcConfig.LocalPort + " -d -f m"
						if configFile.PublicConfig.WifiChip == "realtek" {
							ottIperfUpCmd5 = "echo 1 > /proc/net/rtl88x2cs/wlan0/scan_deny&&" +
								"echo 1 > /proc/net/rtl88x2cs/wlan0/tx_quick_addba_req&&" +
								"cat /proc/net/rtl88x2cs/wlan0/scan_abort&&" +
								"\niperf -s -p" + configFile.PublicConfig.Ott5Port
						} else {
							ottIperfUpCmd5 = "iperf -s -p" + configFile.PublicConfig.Ott5Port
						}

						checkIperfCmd := exec.Command("cmd", "/c", "TASKLIST", "|findstr", "iperf.exe", ">a.txt")
						checkIperfCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
						checkIperfCmd.Run()
						aFileName := "a.txt"
						if _, err := os.Stat(aFileName); err == nil {
							aFileSize := getSize(localPath + "/" + aFileName)
							aFileContent, _ := ReadAll(localPath + "/" + aFileName)
							if aFileSize != 0 && strings.Contains(string(aFileContent), "iperf.exe") {
								killiperfcmd := exec.Command("cmd", "/c", "TASKKILL", "/IM", "iperf.exe", "/F")
								killiperfcmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
								err := killiperfcmd.Run()
								if err == nil {
									gLogger.Println("kill iperf\r")
									UserInput("\n" + ottIperfUpCmd5 + "\r")
									ottIperfUpCmd5 = "iperf -s -p" + configFile.PublicConfig.Ott5Port
									time.Sleep(time.Duration(2) * time.Second)
									go runIperfCmdInPc(pcIperfUpCmd)
									gLogger.Println("5G pcIperfcmd:", ottIperfUpCmd5)
								}

							} else {
								UserInput("\n" + ottIperfUpCmd5 + "\r")
								ottIperfUpCmd5 = "iperf -s -p" + configFile.PublicConfig.Ott5Port
								time.Sleep(time.Duration(2) * time.Second)
								go runIperfCmdInPc(pcIperfUpCmd)
								gLogger.Println("5G pcIperfcmd:", ottIperfUpCmd5)
							}

						} else {
							UserInput("\n" + ottIperfUpCmd5 + "\r")
							ottIperfUpCmd5 = "iperf -s -p" + configFile.PublicConfig.Ott5Port
							time.Sleep(time.Duration(2) * time.Second)
							go runIperfCmdInPc(pcIperfUpCmd)
							gLogger.Println("5G pcIperfcmd:", ottIperfUpCmd5)
						}

					} else {
						//udp
						var pcIperfUpCmd = "cmd /c " + localPath + "/iperf.exe -s -p" +
							configFile.PcConfig.LocalPort + " -m -u"

						ottIperfUpCmd5 = "iperf -c " + localIp + " -p" +
							configFile.PcConfig.LocalPort + " -i " +
							configFile.PublicConfig.Interval + " -t " +
							configFile.PublicConfig.RunSeconds + " -u -b " +
							configFile.Wifi5Config.LimitSpeed5 + "M -d -L " +
							configFile.PublicConfig.Ott5Port

						checkIperfCmd := exec.Command("cmd", "/c", "TASKLIST", "|findstr", "iperf.exe", ">a.txt")
						checkIperfCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
						checkIperfCmd.Run()
						aFileName := "a.txt"
						if _, err := os.Stat(aFileName); err == nil {
							aFileSize := getSize(localPath + "/" + aFileName)
							aFileContent, _ := ReadAll(localPath + "/" + aFileName)
							if aFileSize != 0 && strings.Contains(string(aFileContent), "iperf.exe") {
								killiperfcmd := exec.Command("cmd", "/c", "TASKKILL", "/IM", "iperf.exe", "/F")
								killiperfcmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
								err = killiperfcmd.Run()
								if err == nil {
									gLogger.Println("kill iperf\r")
									go runIperfCmdInPc(pcIperfUpCmd)
									time.Sleep(time.Duration(2) * time.Second)
									UserInput("\n" + ottIperfUpCmd5 + "\r")
									gLogger.Println("5G pcIperfcmd:", ottIperfUpCmd5)
								}
							} else {
								go runIperfCmdInPc(pcIperfUpCmd)
								time.Sleep(time.Duration(2) * time.Second)
								UserInput("\n" + ottIperfUpCmd5 + "\r")
								gLogger.Println("5G pcIperfcmd:", ottIperfUpCmd5)
							}
						} else {
							go runIperfCmdInPc(pcIperfUpCmd)
							time.Sleep(time.Duration(2) * time.Second)
							UserInput("\n" + ottIperfUpCmd5 + "\r")
							gLogger.Println("5G pcIperfcmd:", ottIperfUpCmd5)
						}
						ottIperfUpCmd5 = "iperf -c " + localIp + " -p" +
							configFile.PcConfig.LocalPort + " -i " +
							configFile.PublicConfig.Interval + " -t " +
							configFile.PublicConfig.RunSeconds + " -u -b " +
							configFile.Wifi5Config.LimitSpeed5 + "M"

					}

				}
			} else if ottIperfUpCmd24 != "" && strings.Contains(telnetEditStr, ottIperfUpCmd24) {
				// deal with up result
				if strings.Contains(telnetEditStr, "Server Report:") ||
					configFile.PublicConfig.PackageMethod == "tcp" {
					var iperfResult = dealIperfResultInOtt(configFile.PublicConfig.Ott24Port, telnetEditStr)
					if iperfResult.upRlt == "" || iperfResult.downRlt == "" {
						walk.MsgBox(mw, "2.4G上下行值未得到", "请检查环境", walk.MsgBoxIconWarning)
						return
					}
					if iperfResult.upUnit != "Mbits/sec" || iperfResult.downUnit != "Mbits/sec" {
						walk.MsgBox(mw, "提示", "测试结果偏低请重测", walk.MsgBoxIconWarning)
						return
					}
					upSpeedFloat, errFloat := strconv.ParseFloat(iperfResult.upRlt, 64)
					if errFloat != nil {
						walk.MsgBox(mw, "解析2.4g上行值出错", errFloat.Error(), walk.MsgBoxIconWarning)
					}

					downSpeedFloat, errFloat := strconv.ParseFloat(iperfResult.downRlt, 64)
					if errFloat != nil {
						walk.MsgBox(mw, "解析2.4g下行值出错", errFloat.Error(), walk.MsgBoxIconWarning)
					}

					configFile.Wifi24Config.LimitSpeed24UpRlt = iperfResult.upRlt + iperfResult.upUnit
					configFile.Wifi24Config.LimitSpeed24DownRlt = iperfResult.downRlt + iperfResult.downUnit

					upSpeedCompareFloat, errFloat := strconv.ParseFloat(configFile.Wifi24Config.LimitSpeed24Up, 64)
					if errFloat != nil {
						walk.MsgBox(mw, "解析上行对比时间值出错", errFloat.Error(), walk.MsgBoxIconWarning)
					}
					downSpeedCompareFloat, errFloat := strconv.ParseFloat(configFile.Wifi24Config.LimitSpeed24Down, 64)
					if errFloat != nil {
						walk.MsgBox(mw, "解析下行对比时间值出错", errFloat.Error(), walk.MsgBoxIconWarning)
					}
					if upSpeedFloat >= upSpeedCompareFloat {
						configFile.Wifi24Config.Button24UpIcon = "img/button_check_48px.png"
					} else {
						configFile.Wifi24Config.Button24UpIcon = "img/error_48px.png"
					}

					if downSpeedFloat >= downSpeedCompareFloat {
						configFile.Wifi24Config.Button24DownIcon = "img/button_check_48px.png"
					} else {
						configFile.Wifi24Config.Button24DownIcon = "img/error_48px.png"
					}
					db.Reset()
					killPcIperf()
					if upSpeedFloat >= upSpeedCompareFloat && downSpeedFloat >= downSpeedCompareFloat {
						ottIperfUpCmd24 = ""
						pcTextEdit.SetText("")
						telnetEdit.SetText("")
						if configFile.PublicConfig.WifiChip == "realtek" {
							UserInput("\n" + "echo 0 >/proc/net/rtl88x2cs/wlan0/scan_deny&&\n" +
								ConnectWifiCmd + configFile.Wifi5Config.Ssid5 + "\r")
						} else {
							UserInput("\n" + ConnectWifiCmd + configFile.Wifi5Config.Ssid5 + "\r")
						}
					} else {
						walk.MsgBox(mw, "2.4G测试不通过", "请重新测试", walk.MsgBoxIconWarning)
					}

				} else {
					walk.MsgBox(mw, "提示", "2.4Gwifi断开请检查环境并重测", walk.MsgBoxIconWarning)
				}
			} else if ottIperfUpCmd5 != "" && strings.Contains(telnetEditStr, ottIperfUpCmd5) {
				gLogger.Println("5gresult send package:", configFile.PublicConfig.PackageMethod)
				if strings.Contains(telnetEditStr, "Server Report:") ||
					configFile.PublicConfig.PackageMethod == "tcp" {
					var iperfResult = dealIperfResultInOtt(configFile.PublicConfig.Ott5Port, telnetEditStr)
					gLogger.Println("5gresult:", iperfResult)
					if iperfResult.upRlt == "" || iperfResult.downRlt == "" {
						walk.MsgBox(mw, "5G上下行值未得到", "请检查环境", walk.MsgBoxIconWarning)
						return
					}
					if iperfResult.upUnit != "Mbits/sec" || iperfResult.downUnit != "Mbits/sec" {
						walk.MsgBox(mw, "提示", "测试结果偏低请重测", walk.MsgBoxIconWarning)
						return
					}
					upSpeedFloat, errFloat := strconv.ParseFloat(iperfResult.upRlt, 64)
					if errFloat != nil {
						walk.MsgBox(mw, "解析5g上行值出错", errFloat.Error(), walk.MsgBoxIconWarning)
					}

					downSpeedFloat, errFloat := strconv.ParseFloat(iperfResult.downRlt, 64)
					if errFloat != nil {
						walk.MsgBox(mw, "解析5g下行值出错", errFloat.Error(), walk.MsgBoxIconWarning)
					}

					configFile.Wifi5Config.LimitSpeed5UpRlt = iperfResult.upRlt + iperfResult.upUnit
					configFile.Wifi5Config.LimitSpeed5DownRlt = iperfResult.downRlt + iperfResult.downUnit

					upSpeedCompareFloat, errFloat := strconv.ParseFloat(configFile.Wifi5Config.LimitSpeed5Up, 64)
					if errFloat != nil {
						walk.MsgBox(mw, "解析5g上行对比时间值出错", errFloat.Error(), walk.MsgBoxIconWarning)
					}
					downSpeedCompareFloat, errFloat := strconv.ParseFloat(configFile.Wifi5Config.LimitSpeed5Down, 64)
					if errFloat != nil {
						walk.MsgBox(mw, "解析5g下行对比时间值出错", errFloat.Error(), walk.MsgBoxIconWarning)
					}
					if upSpeedFloat >= upSpeedCompareFloat {
						configFile.Wifi5Config.Button5UpIcon = "img/button_check_48px.png"
					} else {
						configFile.Wifi5Config.Button5UpIcon = "img/error_48px.png"
					}

					if downSpeedFloat >= downSpeedCompareFloat {
						configFile.Wifi5Config.Button5DownIcon = "img/button_check_48px.png"
					} else {
						configFile.Wifi5Config.Button5DownIcon = "img/error_48px.png"
					}

					db.Reset()
					killPcIperf()
					if upSpeedFloat >= upSpeedCompareFloat && downSpeedFloat >= downSpeedCompareFloat {
						ottIperfUpCmd5 = ""

						//setRlt
						// pcTextEdit.SetText("")
						telnetEdit.SetText("")
						setPassRate(true)
						if configFile.PcConfig.ReportData {
							//factory report sn means bobsn
							request := RequestBean{
								Dut_Sn: configFile.PublicConfig.Bobsn,
								Cur_Station:  "WIFI_THRO",
								Status: "PASS",
							}
							jsonRequest, err := json.Marshal(request)
							if err != nil {
								gLogger.Println("生成json字符串错误")
							}
							//var content = "Dut_Sn=" + configFile.PublicConfig.Bobsn + "&Cur_Station=WIFI_THRO&Status=PASS"
							body := mw.httpDo("POST", "http://192.168.2.101:8888/DBATE/UpdateStatus", string(jsonRequest))
							gLogger.Println("response:", body)

							if body != "" {
								var response ResponseBean
								json.Unmarshal([]byte(body), &response)
								gLogger.Println("updatestatus parse is:", response)
								if response.IsSuccess {
									go showTestPassDlg()
									if !PING_DISMISS_DLG {
										PING_DISMISS_DLG = true
										go pingAndDismissDlg()
									}
								} else {
									go walk.MsgBox(mw, "测试完毕但上传服务器失败", "请重测,服务器提示:"+response.Message, walk.MsgBoxIconWarning)
								}
							}

						} else {
							go showTestPassDlg()
							gLogger.Println("showTestPassDlg PING_DISMISS_DLG is:", PING_DISMISS_DLG)
							if !PING_DISMISS_DLG {
								PING_DISMISS_DLG = true
								go pingAndDismissDlg()
								gLogger.Println("pingAndDismissDlg")
							}
						}
						gLogger.Println("write result")
						UserInput("\n" + "echo 1 > /data/local/iperf-test&&chmod 777 /data/local/iperf-test&&sync" + "\r")
					} else {
						// pcTextEdit.SetText("")
						telnetEdit.SetText("")
						setPassRate(false)
						if configFile.PcConfig.ReportData {
							//factory report sn means bobsn
							request := RequestBean{
								Dut_Sn: configFile.PublicConfig.Bobsn,
								Cur_Station:  "WIFI_THRO",
								Status: "FAIL",
							}
							jsonRequest, err := json.Marshal(request)
							if err != nil {
								gLogger.Println("生成json字符串错误")
							}
							//var content = "Dut_Sn=" + configFile.PublicConfig.Bobsn + "&Cur_Station=WIFI_THRO&Status=FAIL"
							body := mw.httpDo("POST", "http://192.168.2.101:8888/DBATE/UpdateStatus", string(jsonRequest))
							gLogger.Println("response:", body)
							if body != "" {
								var response ResponseBean
								json.Unmarshal([]byte(body), &response)
								gLogger.Println("updatestatus parse is:", response)
								if response.IsSuccess {
									go walk.MsgBox(mw, "5G测试不通过", "请重测(已上传服务器)", walk.MsgBoxIconWarning)
								} else {
									go walk.MsgBox(mw, "测试不通过", "服务器提示:"+response.Message, walk.MsgBoxIconWarning)
								}
							}
						} else {
							go walk.MsgBox(mw, "5G测试不通过", "请重新测试", walk.MsgBoxIconWarning)
						}
						if configFile.PublicConfig.WifiChip == "realtek" {
							UserInput("\n" + "echo 0 >/proc/net/rtl88x2cs/wlan0/scan_deny&&" +
								"echo 0 > /data/local/iperf-test&&chmod 777 /data/local/iperf-test&&sync" + "\r")
						} else {
							UserInput("\n" + "echo 0 > /data/local/iperf-test&&chmod 777 /data/local/iperf-test&&sync" + "\r")
						}
					}

				} else {
					walk.MsgBox(mw, "提示", "5Gwifi断开请检查环境并重测", walk.MsgBoxIconWarning)
				}
			}
		}

	}
}

func ReadAll(filePth string) ([]byte, error) {
	f, err := os.Open(filePth)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}

//获取单个文件的大小
func getSize(path string) int64 {
	fileInfo, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	fileSize := fileInfo.Size() //获取size
	gLogger.Println(path+" 的大小为", fileSize, "byte")
	return fileSize
}

func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])

	path, _ := filepath.Abs(file)

	rst := filepath.Dir(path)

	return strings.Replace(rst, "\\", "/", -1)
}

func runIperfCmdInPc(cmd string) {
	pcIperfRlt := exeSysCommand(cmd)
	gLogger.Println("pcIperfResult:", pcIperfRlt)
}

func dealIperfResultInOtt(ottPort, iperfRlt string) IperfResult {
	iperfRlt = strings.Replace(iperfRlt, "\n", "", -1)
	upFlag, downFlag := "", ""
	var iperfResult IperfResult
	iperfResult.upRlt = ""
	iperfResult.upUnit = ""
	kl := strings.Split(iperfRlt, "\r")
	lineNum := len(kl)
	for i := 0; i < lineNum; i++ {
		lineStr := kl[i]

		if strings.Contains(lineStr, "connected with") && strings.Contains(lineStr, localIp) {
			//get up/down flag
			flagRlts := strings.Split(lineStr, " ")
			flagLineNum := len(flagRlts)
			for i := 0; i < flagLineNum; i++ {
				flagItem := flagRlts[i]
				if strings.Contains(flagItem, "]") {
					if strings.Contains(lineStr, ottPort) {
						// up speed
						upFlag = flagItem

					} else if strings.Contains(lineStr, configFile.PcConfig.LocalPort) {
						//down speed
						downFlag = flagItem
					}
				}
			}

			gLogger.Println("upFlag:", upFlag)
			gLogger.Println("downFlag:", downFlag)

		}

		if !strings.Contains(lineStr, "received") &&
			strings.Contains(lineStr, "sec") {
			lineStr := delete_extra_space(lineStr)
			resultStr := strings.Split(lineStr, " ")
			resultLineNum := len(resultStr)
			startPosition := 0
			position := 0
			for i := 0; i < resultLineNum; i++ {
				resultItem := resultStr[i]
				if resultItem == "sec" {
					position = i
				} else if strings.Contains(resultItem, "]") {
					startPosition = i
				}
				gLogger.Println("resultItem,", i, " ", resultItem)
			}
			gLogger.Println("result startPosition", startPosition, "sec position:", position)
			timeStr := ""
			if position == 2 {
				timeStr = resultStr[1]
			} else {
				timeStrArr := resultStr[startPosition+1 : position]

				for _, v := range timeStrArr {
					timeStr += v
				}
			}
			allTimeStr := delete_extra_space(timeStr)
			gLogger.Println("allTimeStr:", allTimeStr)
			firstTime := strings.Split(allTimeStr, "-")[0]
			secondTime := strings.Split(allTimeStr, "-")[1]
			gLogger.Println("firstTime:", firstTime)
			gLogger.Println("secondTime:", secondTime)

			if firstTime == "0.0" && secondTime != "1.0" {
				if upFlag != "" && strings.Contains(lineStr, upFlag) {
					iperfResult.upRlt = resultStr[position+3]
					iperfResult.upUnit = resultStr[position+4]
				} else if downFlag != "" && strings.Contains(lineStr, downFlag) {
					iperfResult.downRlt = resultStr[position+3]
					iperfResult.downUnit = resultStr[position+4]
				}
			}
			gLogger.Println("resultStr:", resultStr)
		}
	}
	upSpeedFloat, errFloat := strconv.ParseFloat(iperfResult.upRlt, 64)
	if errFloat != nil {
		walk.MsgBox(mw, "解析上行值出错", errFloat.Error(), walk.MsgBoxIconWarning)
	}

	downSpeedFloat, errFloat := strconv.ParseFloat(iperfResult.downRlt, 64)
	if errFloat != nil {
		walk.MsgBox(mw, "解析下行值出错", errFloat.Error(), walk.MsgBoxIconWarning)
	}
	if upSpeedFloat < downSpeedFloat {
		var tempValue = iperfResult.upRlt
		iperfResult.upRlt = iperfResult.downRlt
		iperfResult.downRlt = tempValue

		var tempUnit = iperfResult.upUnit
		iperfResult.upUnit = iperfResult.downUnit
		iperfResult.downUnit = tempUnit
	}
	gLogger.Println("iperfResult.upRlt:", iperfResult.upRlt, "iperfResult.upUnit:", iperfResult.upUnit)
	gLogger.Println("iperfResult.downRlt:", iperfResult.downRlt, "iperfResult.downUnit:", iperfResult.downUnit)
	return iperfResult
}

func delete_extra_space(s string) string {
	//删除字符串中的多余空格，有多个空格时，仅保留一个空格
	s1 := strings.Replace(s, "	", " ", -1)       //替换tab为空格
	regstr := "\\s{2,}"                          //两个及两个以上空格的正则表达式
	reg, _ := regexp.Compile(regstr)             //编译正则表达式
	s2 := make([]byte, len(s1))                  //定义字符数组切片
	copy(s2, s1)                                 //将字符串复制到切片
	spc_index := reg.FindStringIndex(string(s2)) //在字符串中搜索
	for len(spc_index) > 0 {                     //找到适配项
		s2 = append(s2[:spc_index[0]+1], s2[spc_index[1]:]...) //删除多余空格
		spc_index = reg.FindStringIndex(string(s2))            //继续在字符串中搜索
	}
	return string(s2)
}

func exeSysCommand(cmdStr string) bool {
	gLogger.Println("exeSysCommand" + cmdStr + "\r")
	args := strings.Split(cmdStr, " ")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		gLogger.Println(os.Stderr, "error=>", err.Error())
		return false
	}
	cmd.Start()

	reader := bufio.NewReader(stdout)

	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			gLogger.Println("pccmd exit=>", err2.Error())
			break
		}
		fmt.Println(line)
		pcTextEdit.AppendText(line)
	}
	cmd.Wait()
	return true
}

func UserInput(str string) {
	strIn := strings.NewReader(str)
	input := bufio.NewReader(strIn)
	in, err := input.ReadBytes('\r')
	if err != nil {
		gLogger.Fatalln(err.Error())
	}
	c.Write(in)
}

func clearRlt() {
	macLineEdit.SetFocus()
	macLineEdit.SetText("")
	configFile.Wifi24Config.LimitSpeed24UpRlt = ""
	configFile.Wifi24Config.LimitSpeed24DownRlt = ""
	configFile.Wifi24Config.Button24UpIcon = "img/logo.ico"
	configFile.Wifi24Config.Button24DownIcon = "img/logo.ico"

	configFile.Wifi5Config.LimitSpeed5UpRlt = ""
	configFile.Wifi5Config.LimitSpeed5DownRlt = ""
	configFile.Wifi5Config.Button5UpIcon = "img/logo.ico"
	configFile.Wifi5Config.Button5DownIcon = "img/logo.ico"
	db.Reset()
	pcTextEdit.SetText("")
	telnetEdit.SetText("")
	burnsnflag = false
	ottIperfUpCmd24 = ""
	ottIperfUpCmd5 = ""
}

func showTestPassDlg() {
	reTestDiaInt := walk.MsgBox(mw, "测试通过", "请更换机顶盒", walk.MsgBoxIconInformation|walk.MsgBoxOK)
	gLogger.Println("dialog int:", reTestDiaInt)
	if reTestDiaInt == 1 {
		// telnetEdit.SetText("")
	}
}

func pingAndDismissDlg() {
	pingresult := utils.PingDisconnect(configFile.PublicConfig.OttEthIp)
	gLogger.Println("pingresult:", pingresult)
	if !pingresult {
		c.Release()
		PING_DISMISS_DLG = false
		handle := win.FindWindow(nil, syscall.StringToUTF16Ptr(strings.ReplaceAll("测试通过", "\x00", "␀")))
		win.SendMessage(handle, win.WM_CLOSE, 0, 0)
		if !configFile.PcConfig.ReportData || configFile.PcConfig.RetestItem{
			gLogger.Println("retest")
			clearRlt()
			gLogger.Println("clearRlt")
			go mw.startIperf("", "")
		}
		gLogger.Println("dialogRlt:", pingresult)
	}
}

func killPcIperf() {
	killiperfcmd := exec.Command("cmd", "/c", "TASKKILL", "/IM", "iperf.exe", "/F")
	killiperfcmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	killiperfcmd.Run()
}

func (mw *MyMainWindow) startIperf(sn, bobSn string) {
	killPcIperf()
	RootCmdPre = ""
	pingresult := utils.PingConnectTimeout(configFile.PublicConfig.OttEthIp, configFile.PcConfig.PingNetTimes)
	gLogger.Println("startIperf pingresult :", pingresult)
	if pingresult {
		var logFilename = "debug.log"
		if _, err := os.Stat(logFilename); err == nil {
			fmt.Println("file exists")
			err := os.Truncate(logFilename, 0)
			if err != nil {
				fmt.Println("file truncate failed", err.Error())
			}
		}
		c = telnet.NewClient(configFile.PublicConfig.OttEthIp, "23")
		err := c.Connect(UserOutput)
		if err != nil {
			gLogger.Println("telnet connect failed")
			time.Sleep(time.Duration(5) * time.Second)
			c.Connect(UserOutput)
		}
		if bobSn != "" {
			burnsnflag = true
		}

		errorProcess := c.Process()
		if errorProcess != nil {
			gLogger.Println("telnet disconnect", errorProcess.Error())
			return
		}
		msg := telnetEdit.Text()
		gLogger.Println("msg is :", msg)
	} else {
		walk.MsgBox(mw, "提示", "网口连接超时", walk.MsgBoxIconWarning)
	}

}

func (mw *MyMainWindow) httpDo(method string, url string, content string) (bodys string) {
	client := &http.Client{}
	var req *http.Request
	var err error
	if method == "GET" {
		req, err = http.NewRequest(method, url, nil)
	} else if method == "POST" {
		// req := `{"name":"junneyang", "age": 88}`
		req_new := bytes.NewBuffer([]byte(content))
		req, err = http.NewRequest(method, url, req_new)
	}

	if err != nil {
		walk.MsgBox(mw, "网络故障", err.Error(), walk.MsgBoxIconWarning)
	}

	if method == "POST" {
		//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := client.Do(req)

	if err != nil {
		walk.MsgBox(mw, "网络故障", err.Error(), walk.MsgBoxIconWarning)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	bodys = string(body)

	if err != nil {
		// handle error
		walk.MsgBox(mw, "网络故障", err.Error(), walk.MsgBoxIconWarning)
		return ""
	}

	return bodys
}

func setPassRate(passFlag bool) {
	passCountStr := configFile.PcConfig.Passcount
	dispassCountStr := configFile.PcConfig.Dispasscount

	if passCountStr == "" {
		passCountStr = "0"
	}
	if dispassCountStr == "" {
		dispassCountStr = "0"
	}
	passCountInt, interr := strconv.Atoi(passCountStr)
	if interr != nil {
		walk.MsgBox(mw, "警告", "配置文件被更改请检查", walk.MsgBoxIconWarning)
	}
	dispassCountInt, interr := strconv.Atoi(dispassCountStr)
	if interr != nil {
		walk.MsgBox(mw, "警告", "配置文件被更改请检查", walk.MsgBoxIconWarning)
	}
	if passFlag {
		passCountInt++
		configFile.PcConfig.Passcount = strconv.Itoa(passCountInt)
	} else {
		dispassCountInt++
		configFile.PcConfig.Dispasscount = strconv.Itoa(dispassCountInt)
	}
	gLogger.Println("config passcount,", configFile.PcConfig.Passcount)
	gLogger.Println("config dispasscount,", configFile.PcConfig.Dispasscount)
	gLogger.Println("config passrate,", configFile.PcConfig.Passrate)
	gLogger.Println("passcount,", passCountInt)
	gLogger.Println("dispassCountInt,", dispassCountInt)
	passrate := passCountInt * 100 / (passCountInt + dispassCountInt)
	gLogger.Println("passrate,", passrate)
	configFile.PcConfig.Passrate = strconv.Itoa(passrate) + "%"
	db.Reset()
}
