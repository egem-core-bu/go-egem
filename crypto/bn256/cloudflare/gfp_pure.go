//  +build !amd64 appengine gccgo
package bn256
import (
	"golang.org/x/sys/cpu"
)

//nolint:varcheck
var hasBMI2 = cpu.X86.HasBMI2
