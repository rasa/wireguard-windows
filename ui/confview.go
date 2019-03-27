/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/conf"
)

type labelTextLine struct {
	label *walk.TextLabel
	text  *walk.LineEdit
}

type interfaceView struct {
	publicKey  *labelTextLine
	listenPort *labelTextLine
	mtu        *labelTextLine
	addresses  *labelTextLine
	dns        *labelTextLine
}

type peerView struct {
	publicKey           *labelTextLine
	presharedKey        *labelTextLine
	allowedIPs          *labelTextLine
	endpoint            *labelTextLine
	persistentKeepalive *labelTextLine
	latestHandshake     *labelTextLine
	transfer            *labelTextLine
}

type ConfView struct {
	*walk.ScrollView
	name      *walk.GroupBox
	interfaze *interfaceView
	peers     map[conf.Key]*peerView

	originalWndProc uintptr
	creatingThread  uint32
}

func (lt *labelTextLine) show(text string) {
	s, e := lt.text.TextSelection()
	lt.text.SetText(text)
	lt.label.SetVisible(true)
	lt.text.SetVisible(true)
	lt.text.SetTextSelection(s, e)
}

func (lt *labelTextLine) hide() {
	lt.text.SetText("")
	lt.label.SetVisible(false)
	lt.text.SetVisible(false)
}

func newLabelTextLine(fieldName string, parent walk.Container) *labelTextLine {
	lt := new(labelTextLine)
	lt.label, _ = walk.NewTextLabel(parent)
	lt.label.SetText(fieldName + ":")
	lt.label.SetTextAlignment(walk.AlignHFarVNear)
	lt.label.SetVisible(false)

	lt.text, _ = walk.NewLineEdit(parent)
	win.SetWindowLong(lt.text.Handle(), win.GWL_EXSTYLE, win.GetWindowLong(lt.text.Handle(), win.GWL_EXSTYLE)&^win.WS_EX_CLIENTEDGE)
	lt.text.SetReadOnly(true)
	lt.text.SetBackground(walk.NullBrush())
	lt.text.SetVisible(false)
	lt.text.FocusedChanged().Attach(func() {
		lt.text.SetTextSelection(0, 0)
	})
	return lt
}

func newInterfaceView(parent walk.Container) *interfaceView {
	iv := &interfaceView{
		newLabelTextLine("Public key", parent),
		newLabelTextLine("Listen port", parent),
		newLabelTextLine("MTU", parent),
		newLabelTextLine("Addresses", parent),
		newLabelTextLine("DNS servers", parent),
	}
	layoutInGrid(iv, parent.Layout().(*walk.GridLayout))
	return iv
}

func newPeerView(parent walk.Container) *peerView {
	pv := &peerView{
		newLabelTextLine("Public key", parent),
		newLabelTextLine("Preshared key", parent),
		newLabelTextLine("Allowed IPs", parent),
		newLabelTextLine("Endpoint", parent),
		newLabelTextLine("Persistent keepalive", parent),
		newLabelTextLine("Latest handshake", parent),
		newLabelTextLine("Transfer", parent),
	}
	layoutInGrid(pv, parent.Layout().(*walk.GridLayout))
	return pv
}

func layoutInGrid(view interface{}, layout *walk.GridLayout) {
	v := reflect.ValueOf(view).Elem()
	for i := 0; i < v.NumField(); i++ {
		lt := (*labelTextLine)(unsafe.Pointer(v.Field(i).Pointer()))
		layout.SetRange(lt.label, walk.Rectangle{0, i, 1, 1})
		layout.SetRange(lt.text, walk.Rectangle{2, i, 1, 1})
	}
}

func (iv *interfaceView) apply(c *conf.Interface) {
	iv.publicKey.show(c.PrivateKey.Public().String())

	if c.ListenPort > 0 {
		iv.listenPort.show(strconv.Itoa(int(c.ListenPort)))
	} else {
		iv.listenPort.hide()
	}

	if c.Mtu > 0 {
		iv.mtu.show(strconv.Itoa(int(c.Mtu)))
	} else {
		iv.mtu.hide()
	}

	if len(c.Addresses) > 0 {
		addrStrings := make([]string, len(c.Addresses))
		for i, address := range c.Addresses {
			addrStrings[i] = address.String()
		}
		iv.addresses.show(strings.Join(addrStrings[:], ", "))
	} else {
		iv.addresses.hide()
	}

	if len(c.Dns) > 0 {
		addrStrings := make([]string, len(c.Dns))
		for i, address := range c.Dns {
			addrStrings[i] = address.String()
		}
		iv.dns.show(strings.Join(addrStrings[:], ", "))
	} else {
		iv.dns.hide()
	}
}

func (pv *peerView) apply(c *conf.Peer) {
	pv.publicKey.show(c.PublicKey.String())

	if !c.PresharedKey.IsZero() {
		pv.presharedKey.show("enabled")
	} else {
		pv.presharedKey.hide()
	}

	if len(c.AllowedIPs) > 0 {
		addrStrings := make([]string, len(c.AllowedIPs))
		for i, address := range c.AllowedIPs {
			addrStrings[i] = address.String()
		}
		pv.allowedIPs.show(strings.Join(addrStrings[:], ", "))
	} else {
		pv.allowedIPs.hide()
	}

	if !c.Endpoint.IsEmpty() {
		pv.endpoint.show(c.Endpoint.String())
	} else {
		pv.endpoint.hide()
	}

	if c.PersistentKeepalive > 0 {
		pv.persistentKeepalive.show(strconv.Itoa(int(c.PersistentKeepalive)))
	} else {
		pv.persistentKeepalive.hide()
	}

	if !c.LastHandshakeTime.IsEmpty() {
		pv.latestHandshake.show(c.LastHandshakeTime.String())
	} else {
		pv.latestHandshake.hide()
	}

	if c.RxBytes > 0 || c.TxBytes > 0 {
		pv.transfer.show(fmt.Sprintf("%s received, %s sent", c.RxBytes.String(), c.TxBytes.String()))
	} else {
		pv.transfer.hide()
	}
}

func newPaddedGroupGrid(parent walk.Container) (group *walk.GroupBox, err error) {
	group, err = walk.NewGroupBox(parent)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			group.Dispose()
		}
	}()
	layout := walk.NewGridLayout()
	layout.SetMargins(walk.Margins{10, 15, 10, 5})
	err = group.SetLayout(layout)
	if err != nil {
		return nil, err
	}
	spacer, err := walk.NewHSpacerFixed(group, 10)
	if err != nil {
		return nil, err
	}
	layout.SetRange(spacer, walk.Rectangle{1, 0, 1, 1})
	return group, nil
}

func NewConfView(parent walk.Container) (*ConfView, error) {
	cv := new(ConfView)
	cv.ScrollView, _ = walk.NewScrollView(parent)
	cv.SetLayout(walk.NewVBoxLayout())
	cv.name, _ = newPaddedGroupGrid(cv)
	cv.interfaze = newInterfaceView(cv.name)
	cv.peers = make(map[conf.Key]*peerView)
	cv.creatingThread = windows.GetCurrentThreadId()
	win.SetWindowLongPtr(cv.Handle(), win.GWLP_USERDATA, uintptr(unsafe.Pointer(cv)))
	cv.originalWndProc = win.SetWindowLongPtr(cv.Handle(), win.GWL_WNDPROC, crossThreadMessageHijack)
	return cv, nil
}

//TODO: choose actual good value for this
const crossThreadUpdate = win.WM_APP + 17

var crossThreadMessageHijack = windows.NewCallback(func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	cv := (*ConfView)(unsafe.Pointer(win.GetWindowLongPtr(hwnd, win.GWLP_USERDATA)))
	if msg == crossThreadUpdate {
		cv.setConfiguration((*conf.Config)(unsafe.Pointer(wParam)))
		return 0
	}
	return win.CallWindowProc(cv.originalWndProc, hwnd, msg, wParam, lParam)
})

func (cv *ConfView) SetConfiguration(c *conf.Config) {
	if cv.creatingThread == windows.GetCurrentThreadId() {
		cv.setConfiguration(c)
	} else {
		cv.SendMessage(crossThreadUpdate, uintptr(unsafe.Pointer(c)), 0)
	}
}

func (cv *ConfView) setConfiguration(c *conf.Config) {
	hasSuspended := false
	suspend := func() {
		if !hasSuspended {
			cv.SetSuspended(true)
			hasSuspended = true
		}
	}
	defer func() {
		if hasSuspended {
			cv.SetSuspended(false)
		}
	}()
	title := "Interface: " + c.Name
	if cv.name.Title() != title {
		cv.name.SetTitle(title)
	}
	cv.interfaze.apply(&c.Interface)
	inverse := make(map[*peerView]bool, len(cv.peers))
	for _, pv := range cv.peers {
		inverse[pv] = true
	}
	for _, peer := range c.Peers {
		if pv := cv.peers[peer.PublicKey]; pv != nil {
			pv.apply(&peer)
			inverse[pv] = false
		} else {
			suspend()
			group, _ := newPaddedGroupGrid(cv)
			group.SetTitle("Peer")
			pv := newPeerView(group)
			pv.apply(&peer)
			cv.peers[peer.PublicKey] = pv
		}
	}
	for pv, remove := range inverse {
		if !remove {
			continue
		}
		k, e := conf.NewPrivateKeyFromString(pv.publicKey.text.Text())
		if e != nil {
			continue
		}
		suspend()
		delete(cv.peers, *k)
		groupBox := pv.publicKey.label.Parent().AsContainerBase().Parent().(*walk.GroupBox)
		groupBox.Parent().Children().Remove(groupBox)
		groupBox.Dispose()
	}
}
