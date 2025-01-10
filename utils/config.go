package utils

type Config struct {
	DebugMode                               bool        `json:"debugMode"`
	GameScale                               float64     `json:"gameScale"`
	MeterToPixelRatio                       float64     `json:"meterToPixelRatio"`
	PixelToMeterRatio                       float64     `json:"pixelToMeterRatio"`
	TileHeight                              float64     `json:"tileHeight"`
	OneFrameTime                            float64     `json:"oneFrameTime"`
	FrameCycle                              int         `json:"frameCycle"`
	MaximumNumberOfLayers                   int         `json:"maximumNumberOfLayers"`
	PlayerAttackRange                       float64     `json:"playerAttackRange"`
	PlayerAttackDamage                      float64     `json:"playerAttackDamage"`
	PlayerAttackAngle                       float64     `json:"playerAttackAngle"`
	MapFirstEntity                          int         `json:"mapFirstEntity"`
	NumberOfMapEntities                     int         `json:"numberOfMapEntities"`
	EnemyFirstEntity                        int         `json:"enemyFirstEntity"`
	NumberOfEnemyEntities                   int         `json:"numberOfEnemyEntities"`
	PlayerEntity                            int         `json:"playerEntity"`
	PlayerAnimation                         int         `json:"playerAnimation"`
	PlayerAcc                               int         `json:"playerAcc"`
	EnemyAcc                                int         `json:"enemyAcc"`
	StartingRoomID                          int         `json:"startingRoomId"`
	InitWidth                               int         `json:"initWidth"`
	InitHeight                              int         `json:"initHeight"`
	BackgroundColor                         string      `json:"backgroundColor"`
	MaxCharacterHP                          float64     `json:"maxCharacterHP"`
	DefaultCharacterHP                      float64     `json:"defaultCharacterHP"`
	DefaultEnemyKnockbackForce              float64     `json:"defaultEnemyKnockbackForce"`
	ApplyKnockback                          bool        `json:"applyKnockback"`
	MaxDungeonDepth                         int         `json:"maxDungeonDepth"`
	StartingPosition                        [2]float64  `json:"startingPosition"`
	SpawnOffset                             float64     `json:"spawnOffset"`
	InvulnerabilityTimeAfterDMG             float64     `json:"invulnerabilityTimeAfterDMG"`
	FullHPColor                             [4]float64  `json:"fullHPColor"`
	LowHPColor                              [4]float64  `json:"lowHPColor"`
	TextTagDefaultSize                      int         `json:"textTagDefaultSize"`
	TextTagDefaultLifetime                  float64     `json:"textTagDefaultLifetime"`
	TextTagDefaultSpeed                     float64     `json:"textTagDefaultSpeed"`
	TextTagDefaultAcceleration              float64     `json:"textTagDefaultAcceleration"`
	TextTagDefaultFadeValue                 int         `json:"textTagDefaultFadeValue"`
	WeaponComponentDefaultDamageAmount      int         `json:"weaponComponentDefaultDamageAmount"`
	WeaponComponentDefaultIsAttacking       bool        `json:"weaponComponentDefaultIsAttacking"`
	WeaponComponentDefaultQueuedAttack      bool        `json:"weaponComponentDefaultQueuedAttack"`
	WeaponComponentDefaultQueuedAttackFlag  bool        `json:"weaponComponentDefaultQueuedAttackFlag"`
	WeaponComponentDefaultIsSwingingForward bool        `json:"weaponComponentDefaultIsSwingingForward"`
	WeaponComponentDefaultIsFacingRight     bool        `json:"weaponComponentDefaultIsFacingRight"`
	WeaponComponentDefaultCurrentAngle      float64     `json:"weaponComponentDefaultCurrentAngle"`
	WeaponComponentDefaultInitialAngle      float64     `json:"weaponComponentDefaultInitialAngle"`
	WeaponComponentDefaultRotationSpeed     float64     `json:"weaponComponentDefaultRotationSpeed"`
	WeaponComponentDefaultSwingDistance     float64     `json:"weaponComponentDefaultSwingDistance"`
	WeaponComponentDefaultRemainingDistance float64     `json:"weaponComponentDefaultRemainingDistance"`
	WeaponComponentDefaultRecoilAmount      float64     `json:"weaponComponentDefaultRecoilAmount"`
	WeaponInteractionDistance               int         `json:"weaponInteractionDistance"`
	EnemyData                               []EnemyData `json:"enemyData"`
	ItemsData                               []ItemData  `json:"itemsData"`
}

type EnemyData struct {
	Type          string        `json:"type"`
	Name          string        `json:"name"`
	HP            float64       `json:"hp"`
	Damage        float64       `json:"damage"`
	TextureData   TextureData   `json:"textureData"`
	CollisionData CollisionData `json:"collisionData"`
}

type TextureData struct {
	TileID    uint32 `json:"tileID"`
	TileSet   string `json:"tileSet"`
	TileLayer int32  `json:"tileLayer"`
}

type CollisionData struct {
	Type    int32   `json:"type"`
	Width   float32 `json:"width"`
	Height  float32 `json:"height"`
	XOffset float32 `json:"xOffset"`
	YOffset float32 `json:"yOffset"`
}

type ItemData struct {
	Name        string      `json:"name"`
	Value       float64     `json:"value"`
	Behaviour   string      `json:"behaviour"`
	TextureData TextureData `json:"textureData"`
}
