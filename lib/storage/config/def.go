package config

type CloudStorageConfig struct {
	Bucket        string `yaml:"bucket"`        //bucket name of s3 or bos
	Path          string `yaml:"path"`          //path in the bucket
	Ak            string `yaml:"ak"`            //access key
	Sk            string `yaml:"sk"`            //secrete key
	Region        string `yaml:"region"`        //region, eg. bj
	Endpoint      string `yaml:"endpoint"`      //endpoint, eg. s3.bj.bcebos.com
	LocalCacheDir string `yaml:"localCacheDir"` //cache directory on local disk
}

func NewCloudStorageConfig() *CloudStorageConfig {
	cStorageConfig := &CloudStorageConfig{}
	cStorageConfig.defaultCloudStorageConfig()
	return cStorageConfig
}

func (c *CloudStorageConfig) defaultCloudStorageConfig() {
	c.Bucket = "xchain-cloud-test"
	c.Path = "node1"
	c.Ak = ""
	c.Sk = ""
	c.Region = "bj"
	c.Endpoint = "s3.bj.bcebos.com"
	c.LocalCacheDir = "./data/cache"
}
