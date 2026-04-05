//go:build cgo && darwin

package appui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void setDockIcon(const void* data, int length) {
	@autoreleasepool {
		NSData* imgData = [NSData dataWithBytes:data length:length];
		NSImage* icon = [[NSImage alloc] initWithData:imgData];
		if (icon != nil) {
			[NSApp setApplicationIconImage:icon];
		}
	}
}
*/
import "C"

import (
	_ "embed"
	"unsafe"
)

//go:embed icon.svg
var iconData []byte

func setAppIcon() {
	if len(iconData) == 0 {
		return
	}
	C.setDockIcon(unsafe.Pointer(&iconData[0]), C.int(len(iconData)))
}
