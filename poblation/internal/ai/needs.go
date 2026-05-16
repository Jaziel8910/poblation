package ai

import (
	"sort"

	"github.com/user/poblation/internal/entities"
)

// NeedType identifies a specific unmet drive.
type NeedType string

const (
	NeedHunger    NeedType = "HUNGER"
	NeedThirst    NeedType = "THIRST"
	NeedSleep     NeedType = "SLEEP"
	NeedSafety    NeedType = "SAFETY"
	NeedBelonging NeedType = "BELONGING"
	NeedEsteem    NeedType = "ESTEEM"
	NeedSex       NeedType = "SEX"
	NeedPower     NeedType = "POWER"
	NeedPurpose   NeedType = "PURPOSE"
)

var allNeedTypes = []NeedType{
	NeedHunger,
	NeedThirst,
	NeedSleep,
	NeedSafety,
	NeedBelonging,
	NeedEsteem,
	NeedSex,
	NeedPower,
	NeedPurpose,
}

// NeedPriority stores ranked need pressure after personality weighting.
type NeedPriority struct {
	Need    NeedType `json:"need"`
	Value   float32  `json:"value"`
	Urgency float32  `json:"urgency"`
	Weight  float32  `json:"weight"`
}

// WorldContext carries world and activity facts needed by the AI systems.
type WorldContext struct {
	ConflictActive             bool    `json:"conflict_active"`
	IsSleeping                 bool    `json:"is_sleeping"`
	AteAmount                  float32 `json:"ate_amount"`
	DrankAmount                float32 `json:"drank_amount"`
	PositiveSocialInteractions int     `json:"positive_social_interactions"`
	NegativeSocialInteractions int     `json:"negative_social_interactions"`
	IsAlone                    bool    `json:"is_alone"`
	RecentEsteemShift          float32 `json:"recent_esteem_shift"`
	EnvironmentalSexualStimuli float32 `json:"environmental_sexual_stimuli"`
	HoursSinceSex              int     `json:"hours_since_sex"`
	HasControl                 bool    `json:"has_control"`
	ActiveGoals                int     `json:"active_goals"`
	CompletedGoals             int     `json:"completed_goals"`
}

// NeedsSystem updates and ranks unmet needs for one Poble.
type NeedsSystem struct {
	Poble *entities.Poble
}

// NewNeedsSystem builds a need system bound to one Poble.
func NewNeedsSystem(poble *entities.Poble) *NeedsSystem {
	return &NeedsSystem{Poble: poble}
}

// Update advances need pressure using world context and recent activity.
func (s *NeedsSystem) Update(deltaHours int, context WorldContext) {
	if s == nil || s.Poble == nil || deltaHours <= 0 {
		return
	}

	needs := &s.Poble.Needs
	hours := float32(deltaHours)

	needs.Hunger = clampPercent(needs.Hunger + (hours * 2.8) - clampPercent(context.AteAmount))
	needs.Thirst = clampPercent(needs.Thirst + (hours * 3.2) - clampPercent(context.DrankAmount))

	if context.IsSleeping {
		needs.Sleep = clampPercent(needs.Sleep - (hours * 12.0))
	} else {
		needs.Sleep = clampPercent(needs.Sleep + (hours * 4.0))
	}

	if context.ConflictActive {
		needs.Safety = clampPercent(needs.Safety + (hours * 6.0))
	} else {
		needs.Safety = clampPercent(needs.Safety - (hours * 1.5))
	}

	socialRelief := float32(context.PositiveSocialInteractions) * 7.0
	socialStress := float32(context.NegativeSocialInteractions) * 4.0
	if context.IsAlone {
		needs.Belonging = clampPercent(needs.Belonging + (hours * 2.5) + socialStress - socialRelief)
	} else {
		needs.Belonging = clampPercent(needs.Belonging + socialStress - socialRelief)
	}

	needs.Esteem = clampPercent(
		needs.Esteem -
			clampSignedPercent(context.RecentEsteemShift) +
			(float32(context.NegativeSocialInteractions) * 3.0) -
			(float32(context.PositiveSocialInteractions) * 1.5),
	)

	timeWithoutSexBoost := minFloat32(float32(maxInt(context.HoursSinceSex, 0))/24.0, 3.0)
	stimulusBoost := clampPercent(context.EnvironmentalSexualStimuli) / 50.0
	sexGain := hours * (0.6 + (s.Poble.Personality.Horniness / 40.0) + timeWithoutSexBoost + stimulusBoost)
	needs.Sex = clampPercent(needs.Sex + sexGain)

	if !context.HasControl {
		needs.Power = clampPercent(needs.Power + (hours * (s.Poble.Personality.Ambition / 30.0)))
	} else {
		needs.Power = clampPercent(needs.Power - (hours * 1.5))
	}

	if context.ActiveGoals <= 0 {
		needs.Purpose = clampPercent(needs.Purpose + (hours * 3.0))
	} else {
		needs.Purpose = clampPercent(needs.Purpose - (hours * 0.8))
	}

	if context.CompletedGoals > 0 {
		completed := float32(context.CompletedGoals)
		needs.Purpose = clampPercent(needs.Purpose - (completed * 14.0))
		needs.Esteem = clampPercent(needs.Esteem - (completed * 8.0))
	}
}

// GetUrgentNeeds returns needs above 70 sorted by weighted urgency.
func (s *NeedsSystem) GetUrgentNeeds() []NeedPriority {
	if s == nil || s.Poble == nil {
		return nil
	}

	priorities := make([]NeedPriority, 0, len(allNeedTypes))
	for _, need := range allNeedTypes {
		value := s.needValue(need)
		if value <= 70.0 {
			continue
		}

		weight := s.needWeight(need)
		urgency := s.weightedUrgency(need, value)
		priorities = append(priorities, NeedPriority{
			Need:    need,
			Value:   value,
			Urgency: urgency,
			Weight:  weight,
		})
	}

	sort.SliceStable(priorities, func(i, j int) bool {
		if priorities[i].Urgency == priorities[j].Urgency {
			return priorities[i].Value > priorities[j].Value
		}
		return priorities[i].Urgency > priorities[j].Urgency
	})

	return priorities
}

// GetDominantNeed returns the strongest weighted need, even below urgent range.
func (s *NeedsSystem) GetDominantNeed() NeedType {
	if s == nil || s.Poble == nil {
		return NeedHunger
	}

	dominant := NeedHunger
	bestUrgency := float32(-1)

	for _, need := range allNeedTypes {
		urgency := s.weightedUrgency(need, s.needValue(need))
		if urgency > bestUrgency {
			bestUrgency = urgency
			dominant = need
		}
	}

	return dominant
}

// SatisfyNeed lowers unmet need pressure by amount.
func (s *NeedsSystem) SatisfyNeed(need NeedType, amount float32) {
	if s == nil || s.Poble == nil {
		return
	}

	if field := s.needField(need); field != nil {
		*field = clampPercent(*field - clampPercent(amount))
	}
}

func (s *NeedsSystem) weightedUrgency(need NeedType, value float32) float32 {
	weight := s.needWeight(need)
	regulation := s.selfRegulationFactor(need)
	return clampPercent((value * 0.35) + (value * weight * regulation))
}

func (s *NeedsSystem) selfRegulationFactor(need NeedType) float32 {
	conscientiousness := s.Poble.Personality.Conscientiousness / 100.0

	switch need {
	case NeedHunger, NeedThirst, NeedSex, NeedBelonging:
		return 1.0 - (conscientiousness * 0.25)
	case NeedSafety, NeedPower, NeedPurpose:
		return 1.0 + (conscientiousness * 0.18)
	default:
		return 1.0 + (conscientiousness * 0.06)
	}
}

func (s *NeedsSystem) needWeight(need NeedType) float32 {
	weight := float32(1.0)
	switch s.Poble.Archetype {
	case entities.ArchetypeRuler:
		switch need {
		case NeedPower:
			weight = 1.45
		case NeedEsteem:
			weight = 1.20
		case NeedHunger:
			weight = 0.75
		case NeedThirst:
			weight = 0.85
		}
	case entities.ArchetypeLover:
		switch need {
		case NeedBelonging:
			weight = 1.35
		case NeedSex:
			weight = 1.25
		case NeedPower:
			weight = 0.85
		}
	case entities.ArchetypeCaretaker:
		switch need {
		case NeedBelonging:
			weight = 1.20
		case NeedPurpose:
			weight = 1.30
		}
	case entities.ArchetypeVillain, entities.ArchetypeSchemer:
		switch need {
		case NeedPower:
			weight = 1.35
		case NeedEsteem:
			weight = 1.15
		}
	case entities.ArchetypeGhost:
		switch need {
		case NeedSafety:
			weight = 1.25
		case NeedBelonging:
			weight = 0.70
		}
	case entities.ArchetypeProphet:
		if need == NeedPurpose {
			weight = 1.45
		}
	case entities.ArchetypeWarrior:
		switch need {
		case NeedSafety:
			weight = 1.25
		case NeedPower:
			weight = 1.10
		}
	}

	if need == NeedSex {
		weight += (s.Poble.Personality.Horniness - 50.0) / 250.0
	}
	if need == NeedPower {
		weight += (s.Poble.Personality.Ambition - 50.0) / 200.0
	}
	if need == NeedBelonging {
		weight += (s.Poble.Personality.Extraversion - 50.0) / 250.0
	}

	return maxFloat32(0.55, weight)
}

func (s *NeedsSystem) needValue(need NeedType) float32 {
	if field := s.needField(need); field != nil {
		return *field
	}
	return 0
}

func (s *NeedsSystem) needField(need NeedType) *float32 {
	if s == nil || s.Poble == nil {
		return nil
	}

	switch need {
	case NeedHunger:
		return &s.Poble.Needs.Hunger
	case NeedThirst:
		return &s.Poble.Needs.Thirst
	case NeedSleep:
		return &s.Poble.Needs.Sleep
	case NeedSafety:
		return &s.Poble.Needs.Safety
	case NeedBelonging:
		return &s.Poble.Needs.Belonging
	case NeedEsteem:
		return &s.Poble.Needs.Esteem
	case NeedSex:
		return &s.Poble.Needs.Sex
	case NeedPower:
		return &s.Poble.Needs.Power
	case NeedPurpose:
		return &s.Poble.Needs.Purpose
	default:
		return nil
	}
}
