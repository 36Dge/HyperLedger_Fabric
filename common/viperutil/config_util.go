package viperutil

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var logger = flogging.MustGetLogger("viperutil")

// ConfigPaths returns the paths from environment and
// defaults which are CWD and /etc/hyperledger/fabric.
func ConfigPaths() []string {
	var paths []string
	if p := os.Getenv("FABRIC_CFG_PATH"); p != "" {
		paths = append(paths, p)
	}
	return append(paths, ".", "/etc/hyperledger/fabric")
}

//configparser holds the configuration file locations.
//it keeps the config file directiory loactions and env variables.
//from the file the config is unmarshalled and stored.
type ConfigParser struct {
	//configuration file to process
	configPaths []string
	configName string
	cofigFile string

	//parsed config
	config map[string]interface{}
}

//new creates a configparser instance
func New()*ConfigParser  {
	return &ConfigParser{
		config: map[string]interface{}{},
	}
}

// AddConfigPaths keeps a list of path to search the relevant
// config file. Multiple paths can be provided.
func (c *ConfigParser) AddConfigPaths(cfgPaths ...string) {
	c.configPaths = append(c.configPaths, cfgPaths...)
}

// SetConfigName provides the configuration file name stem. The upper-cased
// version of this value also serves as the environment variable override
// prefix.
func (c *ConfigParser) SetConfigName(in string) {
	c.configName = in
}

// ConfigFileUsed returns the used configFile.
func (c *ConfigParser) ConfigFileUsed() string {
	return c.configFile
}

// Search for the existence of filename for all supported extensions
func (c *ConfigParser) searchInPath(in string) (filename string) {
	var supportedExts []string = []string{"yaml", "yml"}
	for _, ext := range supportedExts {
		fullPath := filepath.Join(in, c.configName+"."+ext)
		_, err := os.Stat(fullPath)
		if err == nil {
			return fullPath
		}
	}
	return ""
}

// Search for the configName in all configPaths
func (c *ConfigParser) findConfigFile() string {
	paths := c.configPaths
	if len(paths) == 0 {
		paths = ConfigPaths()
	}
	for _, cp := range paths {
		file := c.searchInPath(cp)
		if file != "" {
			return file
		}
	}
	return ""
}

// Get the valid and present config file
func (c *ConfigParser) getConfigFile() string {
	// if explicitly set, then use it
	if c.configFile != "" {
		return c.configFile
	}

	c.configFile = c.findConfigFile()
	return c.configFile
}


//readinconfig reads and ummarshals the config file.
func ( c *configParser) ReadInConfig()error{
	cf := c.getConfigFile()
	logger.Debugf("Attempting to open the config file: %s", cf)
	file, err := os.Open(cf)
	if err != nil {
		logger.Errorf("Unable to open the config file: %s", cf)
		return err
	}
	defer file.Close()

	return c.ReadConfig(file)
}


// ReadConfig parses the buffer and initializes the config.
func (c *ConfigParser) ReadConfig(in io.Reader) error {
	return yaml.NewDecoder(in).Decode(c.config)
}

// Get value for the key by searching environment variables.
func (c *ConfigParser) getFromEnv(key string) string {
	envKey := key
	if c.configName != "" {
		envKey = c.configName + "_" + envKey
	}
	envKey = strings.ToUpper(envKey)
	envKey = strings.ReplaceAll(envKey, ".", "_")
	return os.Getenv(envKey)
}

// Prototype declaration for getFromEnv function.
type envGetter func(key string) string












// EnhancedExactUnmarshal is intended to unmarshal a config file into a structure
// producing error when extraneous variables are introduced and supporting
// the time.Duration type
func (c *ConfigParser) EnhancedExactUnmarshal(output interface{}) error {
	oType := reflect.TypeOf(output)
	if oType.Kind() != reflect.Ptr {
		return errors.Errorf("supplied output argument must be a pointer to a struct but is not pointer")
	}
	eType := oType.Elem()
	if eType.Kind() != reflect.Struct {
		return errors.Errorf("supplied output argument must be a pointer to a struct, but it is pointer to something else")
	}

	baseKeys := c.config
	leafKeys := getKeysRecursively("", c.getFromEnv, baseKeys, eType)

	logger.Debugf("%+v", leafKeys)
	config := &mapstructure.DecoderConfig{
		ErrorUnused:      true,
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			bccspHook,
			mapstructure.StringToTimeDurationHookFunc(),
			customDecodeHook,
			byteSizeDecodeHook,
			stringFromFileDecodeHook,
			pemBlocksFromFileDecodeHook,
			kafkaVersionDecodeHook,
		),
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(leafKeys)
}




















