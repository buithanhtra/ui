// 12 july 2014

package ui

import (
	"fmt"
	"syscall"
	"unsafe"
)

// global messages unique to everything
const (
	msgRequest = c_WM_APP + 1 + iota		// + 1 just to be safe
)

var msgwin uintptr

func uiinit() error {
	if err := initWindows(); err != nil {
		return fmt.Errorf("error initializing package ui on Windows: %v", err)
	}
	if err := makemsgwin(); err != nil {
		return fmt.Errorf("error creating message-only window: %v", err)
	}
	if err := makeWindowWindowClass(); err != nil {
		return fmt.Errorf("error creating Window window class: %v", err)
	}
	return nil
}

func uimsgloop() {
	var msg s_MSG

	for {
		res, err := f_GetMessageW(&msg, hNULL, 0, 0)
		if res < 0 {
			panic(fmt.Errorf("error calling GetMessage(): %v", err))
		}
		if res == 0 {		// WM_QUIT
			break
		}
		// TODO IsDialogMessage()
		f_TranslateMessage(&msg)
		f_DispatchMessageW(&msg)
	}
}

func uistop() {
	// this works fine as documented in modal loops, as modal loops are supposed to repost quit messages (http://blogs.msdn.com/b/oldnewthing/archive/2005/02/22/378018.aspx), and all the Windows internal ones do
	f_PostQuitMessage(0)
}

func issue(req *Request) {
	res, err := f_PostMessageW(
		msgwin,
		msgRequest,
		0,
		t_LPARAM(uintptr(unsafe.Pointer(req))))
	if res == 0 {
		panic(fmt.Errorf("error issuing request: %v", err))
	}
}

const msgwinclass = "gouimsgwin"

func makemsgwin() error {
	var wc s_WNDCLASSW

	wc.lpfnWndProc = syscall.NewCallback(msgwinproc)
	wc.hInstance = hInstance
	wc.hIcon = hDefaultIcon
	wc.hCursor = hArrowCursor
	wc.hbrBackground = c_COLOR_BTNFACE + 1
	wc.lpszClassName = syscall.StringToUTF16Ptr(msgwinclass)
	res, err := f_RegisterClassW(&wc)
	if res == 0 {
		return fmt.Errorf("error registering message-only window class: %v", err)
	}
	msgwin, err = f_CreateWindowExW(
		0,
		wc.lpszClassName,
		syscall.StringToUTF16Ptr("package ui message-only window"),
		0,
		c_CW_USEDEFAULT, c_CW_USEDEFAULT,
		c_CW_USEDEFAULT, c_CW_USEDEFAULT,
		c_HWND_MESSAGE, hNULL, hInstance, nil)
	if msgwin == hNULL {
		return fmt.Errorf("error creating message-only window: %v", err)
	}
	return nil
}

func msgwinproc(hwnd uintptr, uMsg t_UINT, wParam t_WPARAM, lParam t_LPARAM) t_LRESULT {
	switch uMsg {
	case msgRequest:
		req := (*Request)(unsafe.Pointer(uintptr(lParam)))
		perform(req)
		return 0
	default:
		return f_DefWindowProcW(hwnd, uMsg, wParam, lParam)
	}
	panic(fmt.Errorf("message-only window procedure does not return a value for message %d (bug in msgwinproc())", uMsg))
}
