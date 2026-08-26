package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 功能：递归删除当前目录及子目录下所有匹配 z*Field.go 的文件
//       删除完后，自动清理因此产生的空目录（递归向上）
//
// 匹配规则：以 "z" 开头，以 "Field.go" 结尾
// ✅ zabcField.go  z123Field.go  zxxxField.go  zField.go
// ❌ Field.go      zField.txt    abcField.go

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Printf("[ERROR] 获取当前目录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[INFO] 开始扫描目录: %s\n", root)

	// 第一步：递归删除匹配的文件
	deletedFiles, err := deleteMatchedFiles(root)
	if err != nil {
		fmt.Printf("[ERROR] 删除文件时出错: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[INFO] 共删除 %d 个文件\n", deletedFiles)

	// 第二步：递归删除空目录
	deletedDirs, err := deleteEmptyDirs(root)
	if err != nil {
		fmt.Printf("[ERROR] 删除空目录时出错: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[INFO] 共删除 %d 个空目录\n", deletedDirs)

	fmt.Println("[INFO] 清理完成 ✅")
}

// deleteMatchedFiles 递归删除所有匹配 "z*Field.go" 的文件
func deleteMatchedFiles(root string) (int, error) {
	count := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过无法访问的路径
		}
		if info.IsDir() {
			return nil
		}

		name := info.Name()
		if strings.HasPrefix(name, "z") && strings.HasSuffix(name, "Field.go") {
			fmt.Printf("  [DELETE FILE] %s\n", path)
			if err := os.Remove(path); err != nil {
				fmt.Printf("  [WARN] 删除失败: %v\n", err)
				return nil
			}
			count++
		}
		return nil
	})

	return count, err
}

// deleteEmptyDirs 多轮递归删除空目录（从最深层开始）
func deleteEmptyDirs(root string) (int, error) {
	count := 0

	for {
		// 收集所有子目录
		var dirs []string
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() && path != root {
				dirs = append(dirs, path)
			}
			return nil
		})
		if err != nil {
			return count, err
		}

		// 按路径长度降序排列（先删最深的）
		for i := 0; i < len(dirs); i++ {
			for j := i + 1; j < len(dirs); j++ {
				if len(dirs[i]) < len(dirs[j]) {
					dirs[i], dirs[j] = dirs[j], dirs[i]
				}
			}
		}

		deletedThisRound := 0
		for _, dir := range dirs {
			empty, err := isDirEmpty(dir)
			if err != nil || !empty {
				continue
			}
			fmt.Printf("  [DELETE DIR]  %s\n", dir)
			if err := os.Remove(dir); err != nil {
				fmt.Printf("  [WARN] 删除目录失败: %v\n", err)
				continue
			}
			deletedThisRound++
			count++
		}

		// 本轮没有删除任何目录，说明清理完毕
		if deletedThisRound == 0 {
			break
		}
	}

	return count, nil
}

// isDirEmpty 判断目录是否为空
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == nil {
		return false, nil // 非空
	}
	return true, nil // EOF = 空目录
}
