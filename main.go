package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// 默认版本列表
var defaultVersions = []int{
	2802, 2189, 2372, 2545, 2612, 2628, 2699, 2944,
	3095, 3258, 3407, 3442, 3504, 3521, 3570, 3586,
	3717,
}

// Windows API 定义
const resourceNameString = "FX_ASI_BUILD"

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procBeginUpdateResource = kernel32.NewProc("BeginUpdateResourceW")
	procUpdateResource      = kernel32.NewProc("UpdateResourceW")
	procEndUpdateResource   = kernel32.NewProc("EndUpdateResourceW")
)

func boolToUintptr(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

func main() {
	var mw *walk.MainWindow
	var logTE *walk.TextEdit
	var lblPath *walk.Label
	var entryCustom *walk.LineEdit
	var versionParent *walk.Composite
	var currentFilePath string

	checkboxes := make(map[int]*walk.CheckBox)
	addCheckbox := func(ver int, checked bool) {
		if _, exists := checkboxes[ver]; exists {
			return
		}
		cb, _ := walk.NewCheckBox(versionParent)
		cb.SetText(fmt.Sprintf("Version: %d", ver))
		cb.SetChecked(checked)
		checkboxes[ver] = cb
		versionParent.RequestLayout()
	}

	MainWindow{
		AssignTo: &mw,
		Title:    "FiveM ASI Build Injector",
		Size:     Size{Width: 800, Height: 500},
		Layout:   HBox{MarginsZero: true},
		OnDropFiles: func(files []string) {
			if len(files) > 0 {
				path := files[0]
				currentFilePath = path
				lblPath.SetText(fmt.Sprintf("File: %s", filepath.Base(path)))
				logTE.AppendText(fmt.Sprintf("Processing: %s", path) + "\r\n")
			}
		},
		Children: []Widget{
			HSplitter{
				Children: []Widget{
					Composite{
						MinSize: Size{Width: 300},
						Layout:  VBox{},
						Children: []Widget{
							Label{
								AssignTo: &lblPath,
								Text:     "Drag your .asi file here...",
							},
							Composite{
								Layout: Grid{Columns: 2},
								Children: []Widget{
									PushButton{
										Text: "All",
										OnClicked: func() {
											for _, cb := range checkboxes {
												cb.SetChecked(true)
											}
										},
									},
									PushButton{
										Text: "None",
										OnClicked: func() {
											for _, cb := range checkboxes {
												cb.SetChecked(false)
											}
										},
									},
								},
							},
							ScrollView{
								Layout: VBox{},
								Children: []Widget{
									Composite{
										AssignTo: &versionParent,
										Layout:   VBox{MarginsZero: true},
									},
								},
							},
							Composite{
								Layout: Grid{Columns: 2},
								Children: []Widget{
									LineEdit{
										AssignTo:   &entryCustom,
										CueBanner:  "Custom Version (e.g. 4000)",
										ColumnSpan: 1,
									},
									PushButton{
										Text: "Add",
										OnClicked: func() {
											val, err := strconv.Atoi(entryCustom.Text())
											if err == nil && val > 0 {
												addCheckbox(val, true)
												entryCustom.SetText("")
											} else {
												walk.MsgBox(mw, "Error", "Invalid version number.", walk.MsgBoxIconError)
											}
										},
									},
								},
							},
							PushButton{
								Text:    "Inject Build Config",
								MinSize: Size{Height: 40},
								OnClicked: func() {
									if currentFilePath == "" {
										walk.MsgBox(mw, "Info", "Please drag an .asi file first!", walk.MsgBoxIconInformation)
										return
									}
									var targetVersions []int
									for ver, cb := range checkboxes {
										if cb.Checked() {
											targetVersions = append(targetVersions, ver)
										}
									}
									if len(targetVersions) == 0 {
										walk.MsgBox(mw, "Info", "No versions selected.", walk.MsgBoxIconInformation)
										return
									}
									logTE.SetText("")
									logTE.AppendText(fmt.Sprintf("Processing: %s", filepath.Base(currentFilePath)) + "\r\n")
									count, err := injectResources(currentFilePath, targetVersions, func(msg string) {
										logTE.AppendText(msg + "\r\n")
									})
									if err != nil {
										walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
										logTE.AppendText("Error: " + err.Error() + "\r\n")
									} else {
										walk.MsgBox(mw, "Success", fmt.Sprintf("Injected %d versions successfully!", count), walk.MsgBoxIconInformation)
										logTE.AppendText("----------------------\r\n")
										logTE.AppendText("--- Completed ---" + "\r\n")
									}
								},
							},
							VSpacer{},
						},
					},
					Composite{
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Injection Log"},
							TextEdit{
								AssignTo: &logTE,
								ReadOnly: true,
								VScroll:  true,
							},
						},
					},
				},
			},
		},
	}.Create()
	for _, v := range defaultVersions {
		addCheckbox(v, true)
	}
	mw.Run()
}

func injectResources(absPath string, versions []int, logger func(string)) (int, error) {
	pFileName, _ := syscall.UTF16PtrFromString(absPath)
	hUpdate, _, _ := procBeginUpdateResource.Call(
		uintptr(unsafe.Pointer(pFileName)),
		boolToUintptr(false),
	)
	if hUpdate == 0 {
		return 0, fmt.Errorf("failed to open file (locked/permission)")
	}
	dummyData := []byte("dummy")
	pData := uintptr(unsafe.Pointer(&dummyData[0]))
	cbData := uintptr(len(dummyData))
	pNameStr, _ := syscall.UTF16PtrFromString(resourceNameString)
	pNamePtr := uintptr(unsafe.Pointer(pNameStr))
	successCount := 0
	for _, version := range versions {
		pType := uintptr(version)
		pName := pNamePtr
		ret, _, _ := procUpdateResource.Call(
			hUpdate,
			pType,
			pName,
			0,
			pData,
			cbData,
		)
		if ret == 0 {
			logger(fmt.Sprintf("[-] failed: %d", version))
		} else {
			logger(fmt.Sprintf("[+] success: %d", version))
			successCount++
		}
	}
	ret, _, _ := procEndUpdateResource.Call(hUpdate, boolToUintptr(false))
	if ret == 0 {
		return successCount, fmt.Errorf("failed to save file (Disk Error)")
	}
	return successCount, nil
}
