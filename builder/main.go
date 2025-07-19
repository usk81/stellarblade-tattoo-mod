package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

// 設定ファイルの構造体
type Config struct {
	ExportBasePath string `json:"export_base_path"`
	SkinBasePath   string `json:"skin_base_path"`
	TattooBasePath string `json:"tattoo_base_path"`
	Skins          []Skin `json:"skins"`
}

type Skin struct {
	Directory string    `json:"directory"`
	FileName  string    `json:"file_name"`
	IsActive  bool      `json:"is_active"`
	Patterns  []Pattern `json:"patterns"`
}

type Pattern struct {
	ExportFilePath string   `json:"export_file_path"`
	Tattoos        []Tattoo `json:"tattoos"`
}

type Tattoo struct {
	FileName string  `json:"fileName"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	X        int     `json:"x"`
	Y        int     `json:"y"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tattoo-compositor <config.json>")
		os.Exit(1)
	}

	configPath := os.Args[1]
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := processConfig(config); err != nil {
		fmt.Printf("Error processing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Image composition completed successfully!")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

func processConfig(config *Config) error {
	for _, skin := range config.Skins {
		if !skin.IsActive {
			continue
		}

		skinPath := filepath.Join(config.SkinBasePath, skin.Directory, skin.FileName)
		skinImg, err := loadImage(skinPath)
		if err != nil {
			return fmt.Errorf("failed to load skin image %s: %v", skinPath, err)
		}

		for _, pattern := range skin.Patterns {
			if err := processPattern(config, pattern, skinImg); err != nil {
				return fmt.Errorf("failed to process pattern: %v", err)
			}
		}
	}

	return nil
}

func processPattern(config *Config, pattern Pattern, skinImg image.Image) error {
	// スキン画像をベースとして新しい画像を作成
	bounds := skinImg.Bounds()
	compositeImg := image.NewRGBA(bounds)
	draw.Draw(compositeImg, bounds, skinImg, bounds.Min, draw.Src)

	// 各タトゥーを合成
	for _, tattoo := range pattern.Tattoos {
		tattooPath := filepath.Join(config.TattooBasePath, tattoo.FileName)
		tattooImg, err := loadImage(tattooPath)
		if err != nil {
			return fmt.Errorf("failed to load tattoo image %s: %v", tattooPath, err)
		}

		// タトゥー画像をリサイズ
		resizedTattoo := resizeImage(tattooImg, int(tattoo.Width), int(tattoo.Height))

		// タトゥー画像を指定位置に合成
		tattooRect := image.Rect(tattoo.X, tattoo.Y, tattoo.X+resizedTattoo.Bounds().Dx(), tattoo.Y+resizedTattoo.Bounds().Dy())
		draw.Draw(compositeImg, tattooRect, resizedTattoo, resizedTattoo.Bounds().Min, draw.Over)
	}

	// 出力ディレクトリを作成
	outputPath := filepath.Join(config.ExportBasePath, pattern.ExportFilePath)
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %v", outputDir, err)
	}

	// 合成した画像を保存
	if err := saveImage(compositeImg, outputPath); err != nil {
		return fmt.Errorf("failed to save composite image %s: %v", outputPath, err)
	}

	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func saveImage(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func resizeImage(img image.Image, targetWidth, targetHeight int) image.Image {
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// 縦横比を維持しながらリサイズ
	scaleX := float64(targetWidth) / float64(originalWidth)
	scaleY := float64(targetHeight) / float64(originalHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(originalWidth) * scale)
	newHeight := int(float64(originalHeight) * scale)

	// 新しい画像を作成
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// 簡易的なnearest neighbor補間でリサイズ
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)

			// 境界チェック
			if srcX >= originalWidth {
				srcX = originalWidth - 1
			}
			if srcY >= originalHeight {
				srcY = originalHeight - 1
			}

			color := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY)
			newImg.Set(x, y, color)
		}
	}

	return newImg
}
