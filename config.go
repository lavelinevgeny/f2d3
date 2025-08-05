// config.go — глобальная конфигурация приложения
package main

// AppConfig хранит параметры запуска и накопленные результаты
type AppConfig struct {
	SrcDir string
	DstDir string
	// флаги из CLI
	UseLog     bool
	MoveFiles  bool
	NumWorkers int
	// списки для итогового отчёта
	SkipList    []string
	RenamedList []string
}

// cfg доступен из любого файла пакета main
var cfg *AppConfig
