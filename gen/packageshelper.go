package gen

import (
	"golang.org/x/tools/go/packages"
)

func InferPackagePath(dir string) (string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return "", err
	}
	if len(pkgs) == 0 {
		return "", nil
	}
	return pkgs[0].PkgPath, nil
}
