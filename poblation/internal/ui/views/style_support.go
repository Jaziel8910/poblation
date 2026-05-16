package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
)

type Theme struct {
	Background      lipgloss.Color
	Surface         lipgloss.Color
	SurfaceAlt      lipgloss.Color
	Border          lipgloss.Color
	BorderAccent    lipgloss.Color
	BorderDanger    lipgloss.Color
	Text            lipgloss.Color
	TextSoft        lipgloss.Color
	Muted           lipgloss.Color
	Accent          lipgloss.Color
	SecondaryAccent lipgloss.Color
	Warning         lipgloss.Color
	Success         lipgloss.Color
	Danger          lipgloss.Color
	Paper           lipgloss.Color
	Ink             lipgloss.Color
	Gold            lipgloss.Color
}

var DefaultTheme = Theme{
	Background:      lipgloss.Color("#0D0D0D"),
	Surface:         lipgloss.Color("#171717"),
	SurfaceAlt:      lipgloss.Color("#232323"),
	Border:          lipgloss.Color("#2D2D2D"),
	BorderAccent:    lipgloss.Color("#4ECDC4"),
	BorderDanger:    lipgloss.Color("#FF4757"),
	Text:            lipgloss.Color("#E8E8E8"),
	TextSoft:        lipgloss.Color("#D6D0C4"),
	Muted:           lipgloss.Color("#888888"),
	Accent:          lipgloss.Color("#FF6B6B"),
	SecondaryAccent: lipgloss.Color("#4ECDC4"),
	Warning:         lipgloss.Color("#FFD93D"),
	Success:         lipgloss.Color("#6BCB77"),
	Danger:          lipgloss.Color("#FF4757"),
	Paper:           lipgloss.Color("#F3E7C9"),
	Ink:             lipgloss.Color("#2B2116"),
	Gold:            lipgloss.Color("#F2C078"),
}

type LayoutManager struct {
	Width  int
	Height int
}

func (l LayoutManager) IsSinglePanel() bool {
	return l.Width < 80
}

func (l LayoutManager) IsCompactHeight() bool {
	return l.Height < 24
}

func (l LayoutManager) IsTriplePanel() bool {
	return l.Width >= 120
}

func (l LayoutManager) MainPanelWidths() (int, int) {
	total := maxInt(44, l.Width)
	mapWidth := int(float64(total) * 0.4)
	if mapWidth < 22 {
		mapWidth = 22
	}
	if mapWidth > total-20 {
		mapWidth = total - 20
	}
	return mapWidth, total - mapWidth
}

func (l LayoutManager) TriplePanelWidths() (int, int, int) {
	total := maxInt(72, l.Width)
	mapWidth := maxInt(24, int(float64(total)*0.34))
	mindWidth := maxInt(24, int(float64(total)*0.24))
	feedWidth := total - mapWidth - mindWidth
	if feedWidth < 24 {
		feedWidth = 24
		mindWidth = maxInt(22, total-mapWidth-feedWidth)
	}
	return mapWidth, mindWidth, total - mapWidth - mindWidth
}

var (
	backgroundColor = DefaultTheme.Background
	surfaceColor    = DefaultTheme.Surface
	borderColor     = DefaultTheme.Border
	primaryColor    = DefaultTheme.Text
	accentColor     = DefaultTheme.Accent
	secondaryColor  = DefaultTheme.SecondaryAccent
	mutedColor      = DefaultTheme.Muted
	warningColor    = DefaultTheme.Warning
	successColor    = DefaultTheme.Success
	dangerColor     = DefaultTheme.Danger

	HeaderStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryAccent).
			Bold(true)

	SubheaderStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Accent).
			Bold(true)

	BodyStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Text)

	MutedStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Muted)

	AccentStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Accent).
			Bold(true)

	DangerStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Danger).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Success).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Warning).
			Bold(true)

	BorderNormal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DefaultTheme.Border).
			Background(DefaultTheme.Background).
			Foreground(DefaultTheme.Text)

	BorderAccent = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DefaultTheme.BorderAccent).
			Background(DefaultTheme.Background).
			Foreground(DefaultTheme.Text)

	BorderDanger = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(DefaultTheme.BorderDanger).
			Background(DefaultTheme.Surface).
			Foreground(DefaultTheme.Text)

	mutedStyle = MutedStyle
)

var EventIcons = map[events.EventType]string{
	events.EventBirth:                "◌",
	events.EventBirthday:             "✦",
	events.EventPregnancy:            "◍",
	events.EventDeathNatural:         "✝",
	events.EventDeathAccident:        "✝",
	events.EventDeathMurder:          "✝",
	events.EventSuicide:              "✝",
	events.EventIllnessOnset:         "✚",
	events.EventRecovery:             "✦",
	events.EventMentalBreakdown:      "◎",
	events.EventRevelation:           "◉",
	events.EventDecisionPoint:        "◇",
	events.EventSexualEncounter:      "♥",
	events.EventKinkDiscovery:        "♥",
	events.EventAffairStart:          "♥",
	events.EventAffairEnd:            "♡",
	events.EventObsessionPeak:        "◎",
	events.EventStalking:             "◉",
	events.EventRestrainingOrder:     "⛔",
	events.EventFightVerbal:          "!",
	events.EventFightPhysical:        "⚔",
	events.EventWarDeclaration:       "⚑",
	events.EventPeaceTreaty:          "☷",
	events.EventMarriage:             "∞",
	events.EventDivorce:              "÷",
	events.EventAdoption:             "◍",
	events.EventBetrayalRevealed:     "✦",
	events.EventForgiveness:          "◌",
	events.EventExile:                "⇢",
	events.EventRumourSpread:         "≈",
	events.EventGossipChain:          "≈",
	events.EventPublicHumiliation:    "※",
	events.EventParty:                "♪",
	events.EventFuneral:              "✝",
	events.EventRitual:               "◈",
	events.EventReligionFounded:      "◈",
	events.EventElection:             "☑",
	events.EventCoup:                 "⚠",
	events.EventRevolution:           "⚠",
	events.EventEarthquake:           "▣",
	events.EventStorm:                "☁",
	events.EventDrought:              "☼",
	events.EventPlague:               "☣",
	events.EventIslandDiscovery:      "⌂",
	events.EventResourceDepletion:    "◻",
	events.EventTechDiscovered:       "⚙",
	events.EventBuildingCollapsed:    "▥",
	events.EventAnimalAttack:         "▲",
	events.EventFire:                 "♨",
	events.EventFlood:                "≈",
	events.EventTradeEstablished:     "¤",
	events.EventMonopolyFormed:       "¤",
	events.EventTheft:                "◇",
	events.EventDebt:                 "¤",
	events.EventInheritance:          "¤",
	events.EventGamblingWin:          "✦",
	events.EventGamblingLoss:         "✕",
	events.EventGamblingResult:       "◇",
	events.EventGenerationEnd:        "☾",
	events.EventLastPersonAlive:      "◉",
	events.EventCivilizationCollapse: "▤",
	events.EventPopulationMilestone:  "◌",
	events.EventEraChange:            "⇡",
	events.EventTeaching:             "✎",
	events.EventNicknameRevealed:     "◉",
}

var ColorForRelationType = map[entities.RelationshipType]lipgloss.Color{
	entities.RelationshipStranger:            mutedColor,
	entities.RelationshipAcquaintance:        lipgloss.Color("#A1A1A1"),
	entities.RelationshipFriend:              secondaryColor,
	entities.RelationshipBestFriend:          successColor,
	entities.RelationshipRival:               warningColor,
	entities.RelationshipEnemy:               dangerColor,
	entities.RelationshipLover:               accentColor,
	entities.RelationshipSpouse:              lipgloss.Color("#F28F8F"),
	entities.RelationshipExSpouse:            lipgloss.Color("#C77878"),
	entities.RelationshipParent:              lipgloss.Color("#9EC1A3"),
	entities.RelationshipChild:               lipgloss.Color("#9EC1A3"),
	entities.RelationshipSibling:             lipgloss.Color("#A7D8DE"),
	entities.RelationshipFamily:              lipgloss.Color("#A7D8DE"),
	entities.RelationshipMentor:              lipgloss.Color("#B5A3FF"),
	entities.RelationshipStudent:             lipgloss.Color("#B5A3FF"),
	entities.RelationshipBoss:                warningColor,
	entities.RelationshipEmployee:            lipgloss.Color("#CBBE8D"),
	entities.RelationshipNeighbor:            lipgloss.Color("#89A6B1"),
	entities.RelationshipAlly:                successColor,
	entities.RelationshipBetrayer:            dangerColor,
	entities.RelationshipObsession:           lipgloss.Color("#FF8FB1"),
	entities.RelationshipCrush:               lipgloss.Color("#FFB3C1"),
	entities.RelationshipFriendsWithBenefits: lipgloss.Color("#F6A6B2"),
	entities.RelationshipCaretaker:           lipgloss.Color("#8FD3B6"),
	entities.RelationshipDependent:           lipgloss.Color("#8FD3B6"),
	entities.RelationshipNemesis:             lipgloss.Color("#FF7A7A"),
	entities.RelationshipToxicAttraction:     lipgloss.Color("#FF9671"),
	entities.RelationshipCodependent:         lipgloss.Color("#F4A261"),
	entities.RelationshipSecretObsession:     lipgloss.Color("#D17B88"),
	entities.RelationshipComplicated:         lipgloss.Color("#C0B283"),
}

var EmojiForMood = map[entities.MoodType]string{
	entities.MoodHappy:     "◕",
	entities.MoodContent:   "○",
	entities.MoodNeutral:   "·",
	entities.MoodAnxious:   "◌",
	entities.MoodSad:       "◔",
	entities.MoodAngry:     "▲",
	entities.MoodDepressed: "◍",
	entities.MoodEuphoric:  "✦",
	entities.MoodObsessive: "◎",
	entities.MoodNumb:      "□",
}

var EventTextStyles = buildEventTextStyles()

func buildEventTextStyles() map[events.EventType]lipgloss.Style {
	result := map[events.EventType]lipgloss.Style{}
	apply := func(style lipgloss.Style, values ...events.EventType) {
		for _, value := range values {
			result[value] = style
		}
	}

	apply(DangerStyle,
		events.EventDeathNatural, events.EventDeathAccident, events.EventDeathMurder, events.EventSuicide,
		events.EventFightPhysical, events.EventWarDeclaration, events.EventCoup, events.EventRevolution,
		events.EventFire, events.EventFlood, events.EventPlague, events.EventAnimalAttack,
	)
	apply(SuccessStyle,
		events.EventBirth, events.EventRecovery, events.EventMarriage, events.EventAdoption,
		events.EventPeaceTreaty, events.EventTechDiscovered, events.EventTradeEstablished,
	)
	apply(WarningStyle,
		events.EventIllnessOnset, events.EventMentalBreakdown, events.EventBetrayalRevealed,
		events.EventDivorce, events.EventAffairStart, events.EventAffairEnd, events.EventPublicHumiliation,
		events.EventResourceDepletion, events.EventDebt, events.EventGamblingLoss, events.EventEraChange,
	)

	for eventType := range EventIcons {
		if _, ok := result[eventType]; !ok {
			result[eventType] = MutedStyle
		}
	}
	return result
}

func eventIcon(eventType events.EventType) string {
	if value, ok := EventIcons[eventType]; ok {
		return value
	}
	return "·"
}

func eventStyle(eventType events.EventType) lipgloss.Style {
	if style, ok := EventTextStyles[eventType]; ok {
		return style
	}
	return MutedStyle
}

func relationColor(relation entities.RelationshipType) lipgloss.Color {
	if value, ok := ColorForRelationType[relation]; ok {
		return value
	}
	return mutedColor
}

func moodEmoji(mood entities.MoodType) string {
	if value, ok := EmojiForMood[mood]; ok {
		return value
	}
	return "·"
}

func panelStyle(kind string) lipgloss.Style {
	switch kind {
	case "accent":
		return BorderAccent.Padding(0, 1)
	case "danger":
		return BorderDanger.Padding(0, 1)
	default:
		return BorderNormal.Padding(0, 1)
	}
}

func compactHint(layout LayoutManager, panels []string, active int) string {
	if !layout.IsSinglePanel() {
		return ""
	}
	name := panels[clampInt(active, 0, len(panels)-1)]
	return MutedStyle.Render(fmt.Sprintf("TAB cambia panel · viendo %s", strings.ToUpper(name)))
}
