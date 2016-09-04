// Gdriver is a simple, generic driver registration and manipulation library.
// You can add any number of grouped drivers together, get name lists, help
// text (short and long) and test for registered drivers.
//
// The driver class you register must contain two functions:
//		New()		This must return an instance of the actual driver class. It can have any type of interface
//					but it can't accept any parameters and must only return the real driver class
//
//		Indentity(lvl)
//					This must accept one parameter giving the level of detail for the ID and return a
//					string that is suitable for printing on stdout.
//
// Example:
//
//
//	type MyDriver	struct{....}							Your real driver that performs the functions
//
//	type RegisterMyDriver {}								The registration function
//	func ( r *RegisterMyDriver ) New() *MyDriver {
//			return &MyDriver{}
//	}
//	func ( r *RegisterMyDriver) Identity( id int ) string {
//			select id {
//				gdriver.IdentityShort:
//					return "Interface of some sort"
//				gdriver.IdentityLong:
//					return "This package is a driver for interfacing ...."
//			}
//			return "MyDriver"  // default to name
//	}
//
//
// YOu would need to register the function, possibly in an init() routine:
//	func init(){
//		driver.Register("somegroup", &RegisterMyDriver{} )
//	}
//
// When you need a driver, you  then call New()
//	driver := gdriver.New( "somegroup","MyDriver")

// Additionally, you can use NewDefault("somegroup") if you have registered
// a default driver OR if you only have one driver
//			gdriver.Default( "somegroup","MyDriver" )
//
// Another approach is to unify the whole driver class into one:
//
//	type MyDriver	struct{...}
//	func ( r *MyDriver ) New() *MyDriver {
//			return &MyDriver{}
//	}
//  func ( r *MyDriver ) Identity( id int ) string {....}
//  Then, add in all of the driver-specific functions.
//
package gdriver

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// These constants allow you to return differing identity strings. NAME should
// be one word, for example "Bcrypt". SHORT should return a short help text:
// "Bcrypt - strong hash function" and LONG should produce a very long text
// describing what it is and how it works.
// IdentityName is what is used to provide the identity of the driver and is used
// for matching requests. IdentityName will ignore case when matching.
// IdentityShort should be a single word or short phrase to identify a driver
// IdentityLong should be a longer help message given detail about a driver
const (
	IdentityName = iota
	IdentityShort
	IdentityLong
)

// IdentityUnknown is a standard string that can be used to indicate the driver or
// group is unknown
const IdentityUnknown = "unknown"

// DefaultSelection is a string indicating this is the default driver
const DefaultSelection = `_*_`

// NameSeparator is a simple string indicating a separation between group and driver anem
const NameSeparator = "."

// New is the function that is called to allocate a new driver. The return will
// need to be cast to the type desired:
//		drive, err := gdriver.New( groupName , driverName )
//      if ! err {
//				... process error ...
//		}
//		driverClass := drive.(YourDriverType)
//
// To shorten the call, you can use:
//		driverClass := gdriver.NewMust( groupName, driverName ).(YourDriverType)
// This will cause a panic if the driver doesn't exist.
//

// DriverInterface is the class requirements for calling register.
type DriverInterface interface {
	New() interface{}
	Identity(int) string
}

// driverMember is used internally to hold information about a driver. This helps
// make things a bit simpler. Note that the groupname and driver name will be stored
// as passed without case conversion. The key, however, is groupname.drivername
// in the library and the key is created by using the function libraryKey(...)
type driverMember struct {
	Group     string
	Name      string
	Driver    DriverInterface
	Default   bool
	Singleton interface{}
}

// The main storage, global, for the driver data. The mutex must be locked/unlocked
// before any action in order to stop goroutines from colliding.
var (
	driverLibrary map[string]*driverMember
	driverMu      sync.Mutex
	isInitialised bool
)

// Register a new driver into a group. The driver must be able to resolve the name
// by the Identity() function. Internally, all groups and drivers are stored in
// lowercase and separated by a period: e.g. "SQL","MySQL" will get stored with the
// tag of "sql.mysql"
func Register(groupName string, newDriver DriverInterface) {

	driverName := newDriver.Identity(IdentityName)
	if driverName == "" || driverName == DefaultSelection {
		panic("Driver did not supply a valid name")
	}

	member := &driverMember{
		Name:    driverName,
		Group:   groupName,
		Driver:  newDriver,
		Default: false}

	driverMu.Lock()
	defer driverMu.Unlock()
	if !isInitialised {
		driverLibrary = make(map[string]*driverMember)
	}
	driverKey := libraryKey(groupName, driverName)

	if _, ok := driverLibrary[driverKey]; ok {
		panic("Driver '" + driverKey + "' already exists")
	}
	driverLibrary[driverKey] = member
	isInitialised = true

	return
}

// IsRegistered will determine if the group and driver name is valid
func IsRegistered(groupName, driverName string) (found bool) {
	if !isInitialised {
		return false
	}

	driverMu.Lock()
	defer driverMu.Unlock()

	_, found = driverLibrary[libraryKey(groupName, driverName)]
	return found
}

// GetDriver will return the low-level, registered driver for a named group/interface
// If you want the default name you must look it up with the GetDefaultName function.
// This allows you to call New and ID from anywhere you want.
func GetDriver(groupName, driverName string) (DriverInterface, error) {
	driverMu.Lock()
	defer driverMu.Unlock()
	if driverInstance, ok := driverLibrary[libraryKey(groupName, driverName)]; ok {
		return driverInstance.Driver, nil
	}
	return nil, errors.New("Invalid driver: " + groupName + ":" + driverName)
}

// NewDefault is a wrapper to a default registered driver for a group.
func NewDefault(groupName string) (interface{}, error) {
	return New(groupName, DefaultSelection)
}

// MustNewDefault is a wrapper to a default, required register driver
func MustNewDefault(groupName string) interface{} {
	return MustNew(groupName, DefaultSelection)
}

// New will call the driver's New() function and return a new instance of the driver class
func New(groupName, driverName string) (interface{}, error) {
	if !isInitialised {
		return nil, errors.New("Library is not initialised")
	}

	if driverName == DefaultSelection {
		return newDefault(groupName)
	}

	driverMu.Lock()
	defer driverMu.Unlock()

	return findDriver(groupName, driverName)
}

// NewMust is a simple wrapper around the New function that will cause an error if there
// is no driver found for a name.
func MustNew(groupName, driverName string) interface{} {
	d, err := New(groupName, driverName)
	if err != nil {
		panic(err.Error())
	}
	return d
}

// newDefault will find a driver in the group that is either unique or is marked as a default. This is called by New()
// when the driverName indicates default
func newDefault(groupName string) (interface{}, error) {

	driverMu.Lock()
	defer driverMu.Unlock()

	return findDefaultDriver(groupName)
}

// Default will make sure that only ONE driver is made a default
func Default(groupName, driverName string) (found bool) {
	// It must be initialised AND the driver name can't be what we use as a default driver name
	if !isInitialised || driverName == DefaultSelection {
		return false
	}

	driverMu.Lock()
	defer driverMu.Unlock()

	name := libraryKey(groupName, driverName)

	if _, found = driverLibrary[name]; found {
		driverLibrary[name].Default = true
	}
	return found
}

func GetDefaultName(groupName string) (string, error) {
	driverMu.Lock()
	defer driverMu.Unlock()

	lname := strings.ToLower(groupName) + NameSeparator
	for key, driverInstance := range driverLibrary {
		if strings.HasPrefix(key, lname) {
			if driverInstance.Default {
				return driverInstance.Name, nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf("No default driver set for %d", groupName))
}

// Help will return a help string at the level requested
func Help(groupName, driverName string, level int) string {
	if !isInitialised {
		return ""
	}

	driverMu.Lock()
	defer driverMu.Unlock()

	if driver, ok := driverLibrary[libraryKey(groupName, driverName)]; ok {
		return driver.Driver.Identity(level)
	}
	return ""
}

// GroupList will return a complete list of all the groups that have been registered and the number of entries
// for that type.
func ListGroup() map[string]int {
	var groupNames map[string]int
	groupNames = make(map[string]int)

	if isInitialised {
		driverMu.Lock()
		defer driverMu.Unlock()

		for _, driverEntry := range driverLibrary {
			groupId := driverEntry.Group
			if _, ok := groupNames[groupId]; ok {
				groupNames[groupId] = groupNames[groupId] + 1
			} else {
				groupNames[groupId] = 1
			}
		}
	}
	return groupNames
}

// libraryKey generates a single key from the group name and driver name. Both
// are forced to lower case in order to find them. This can be changed to allow
// mixed case names as required.
func libraryKey(groupName, driverName string) string {
	return strings.ToLower(groupName) + NameSeparator + strings.ToLower(driverName)
}

func findDriver(groupName, driverName string) (interface{}, error) {
	if driverInstance, ok := driverLibrary[libraryKey(groupName, driverName)]; ok {
		return driverInstance.Driver.New(), nil
	}
	return nil, errors.New("Invalid driver: " + groupName + ":" + driverName)
}

func findDefaultDriver(groupName string) (interface{}, error) {
	lname := strings.ToLower(groupName) + NameSeparator
	for key, driverInstance := range driverLibrary {
		if strings.HasPrefix(key, lname) {
			if driverInstance.Default || len(driverLibrary) == 1 {
				return driverInstance.Driver.New(), nil
			}
		}
	}
	return nil, errors.New(fmt.Sprintf("No default driver set for %d", groupName))
}
