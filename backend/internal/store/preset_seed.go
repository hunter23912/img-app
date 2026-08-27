package store

import "database/sql"

var starterPromptPresets = []promptPreset{
	{ID: "starter-remove-text-watermark", Name: "去除水印", Scope: "edit", Prompt: "移除图片中的所有文字、Logo 和水印，并根据周围纹理、光影和透视自然补全缺失区域。"},
	{ID: "starter-remove-face-sticker-mosaic", Name: "去脸部遮挡", Scope: "edit", Prompt: "去除人物脸部的贴图、表情贴纸、马赛克和遮挡元素，根据脸部轮廓、五官、皮肤纹理、光影和透视自然还原。"},
	{ID: "starter-enhance-quality", Name: "提升画质", Scope: "edit", Prompt: "超清画质，提升图片清晰度、细节和质感，修复模糊、噪点、压缩痕迹和锯齿。"},
	{ID: "starter-reduce-gpt-image-texture", Name: "减少鱼鳞纹", Scope: "edit", Prompt: "减少鱼鳞状纹理、碎片状褶皱、重复片状细节和不自然的表面噪声。"},
	{ID: "starter-natural-body-details", Name: "人物细节协调", Scope: "edit", Prompt: "优化人物的手部、手指、脚部、脚趾、四肢比例、头发和发丝结构，使其自然、合理、协调并符合人体逻辑。保持人物姿态、表情、服装、构图和原有画风不变，避免新增肢体、手指粘连或发丝杂乱。"},
	{ID: "starter-anime-clean-lines", Name: "漫画清线", Scope: "edit", Prompt: "保持或转换为干净的二次元漫画画风，统一线稿粗细和结构，减少画面中杂乱、断裂、重复和无意义的线条，整理背景与服装细节，强化主体轮廓，保持构图、人物特征、色彩和主要内容不变。"},
	{ID: "female-generate", Name: "角色生成", Scope: "generate", Prompt: "生成一张高冷御姐图，长发，穿着网红网纱披肩式防晒衣，衣服具有层次感，垂坠感，人物手脚合理协调流畅，符合逻辑，减少画面褶皱噪点，保持干净。"},
}

func seedPromptPresets(tx *sql.Tx) error {
	for _, preset := range starterPromptPresets {
		now := nowString()
		if _, err := tx.Exec(`
			INSERT INTO prompt_presets (id, name, prompt, scope, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO NOTHING
		`, preset.ID, preset.Name, preset.Prompt, preset.Scope, now, now); err != nil {
			return err
		}
	}
	return nil
}
