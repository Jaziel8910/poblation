package world

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
)

// TradeoOffer stores what one Poble gives and wants in a barter or market exchange.
type TradeoOffer struct {
	GivingItems  []Item `json:"giving_items"`
	WantingItems []Item `json:"wanting_items"`
}

// TradeEvent records one finished exchange.
type TradeEvent struct {
	ID           string      `json:"id"`
	Day          GameTime    `json:"day"`
	TraderAID    string      `json:"trader_a_id"`
	TraderBID    string      `json:"trader_b_id"`
	Offer        TradeoOffer `json:"offer"`
	Accepted     bool        `json:"accepted"`
	Haggled      bool        `json:"haggled"`
	DramaScore   int         `json:"drama_score"`
	Summary      string      `json:"summary"`
	CurrencyUsed bool        `json:"currency_used"`
}

// TradeResult stores the outcome of one trade attempt.
type TradeResult struct {
	Accepted          bool   `json:"accepted"`
	Haggled           bool   `json:"haggled"`
	AdvantageTo       string `json:"advantage_to"`
	RelationshipDelta int    `json:"relationship_delta"`
	Summary           string `json:"summary"`
}

// InheritanceGrant stores one inheritance transfer.
type InheritanceGrant struct {
	RecipientID     string   `json:"recipient_id"`
	Money           int      `json:"money"`
	Items           []Item   `json:"items"`
	RevealedSecrets []string `json:"revealed_secrets"`
	ConflictRisk    int      `json:"conflict_risk"`
}

// TheftEvent stores the result of a theft attempt.
type TheftEvent struct {
	ID                string `json:"id"`
	ThiefID           string `json:"thief_id"`
	VictimID          string `json:"victim_id"`
	StolenItems       []Item `json:"stolen_items"`
	StolenMoney       int    `json:"stolen_money"`
	Discovered        bool   `json:"discovered"`
	LegalConsequence  string `json:"legal_consequence"`
	ResentmentCreated int    `json:"resentment_created"`
	Summary           string `json:"summary"`
}

// GamblingOutcome stores one participant result inside a gambling session.
type GamblingOutcome struct {
	PobleID  string `json:"poble_id"`
	Stake    int    `json:"stake"`
	Winnings int    `json:"winnings"`
	Net      int    `json:"net"`
}

// GamblingSession stores a procedural betting event.
type GamblingSession struct {
	ID           string            `json:"id"`
	Day          GameTime          `json:"day"`
	Participants []string          `json:"participants"`
	Pot          int               `json:"pot"`
	WinnerID     string            `json:"winner_id"`
	Outcomes     []GamblingOutcome `json:"outcomes"`
	DramaScore   int               `json:"drama_score"`
	Summary      string            `json:"summary"`
}

// EconomyManager tracks emergent economic state for the world.
type EconomyManager struct {
	world        *World
	Prices       map[ResourceType]int `json:"prices"`
	TradeHistory []TradeEvent         `json:"trade_history"`
	HasCurrency  bool                 `json:"has_currency"`
}

// NewEconomyManager builds a runtime economy manager.
func NewEconomyManager(world *World) *EconomyManager {
	manager := &EconomyManager{
		world:        world,
		Prices:       defaultPrices(),
		TradeHistory: []TradeEvent{},
	}
	manager.syncCurrency()
	return manager
}

// DailyResourceUpdate consumes and produces resources based on population and work.
func DailyResourceUpdate(world *World) {
	if world == nil {
		return
	}

	if world.GetPopulation() == 0 {
		return
	}

	for _, island := range world.Islands {
		if island == nil || !island.IsDiscovered {
			continue
		}

		islandPopulation := len(island.Pobles)
		if islandPopulation == 0 {
			continue
		}
		consumption := map[ResourceType]int{
			ResourceFood:  maxInt(1, islandPopulation),
			ResourceWater: maxInt(1, islandPopulation),
			ResourceWood:  maxInt(0, islandPopulation/5),
		}
		for resource, amount := range consumption {
			island.Resources[resource] = maxInt(0, island.Resources[resource]-amount)
		}

		produced := produceResources(island, world)
		for resource, amount := range produced {
			island.Resources[resource] += amount
		}

		applyScarcityPressure(island, world)
	}

	population := world.GetPopulation()
	if totalResources(world)[ResourceFood] > population*10 && totalResources(world)[ResourceWater] > population*10 {
		for _, island := range world.Islands {
			if island != nil && island.IsDiscovered {
				island.Resources[ResourceKnowledge] += 1
			}
		}
	}

	world.syncState()
}

// Trade attempts a barter deal between two Pobles.
func (m *EconomyManager) Trade(aID string, bID string, offer TradeoOffer) TradeResult {
	if m == nil || m.world == nil {
		return TradeResult{Summary: "trade failed because the world economy was missing"}
	}
	a := m.world.GetPoble(aID)
	b := m.world.GetPoble(bID)
	if a == nil || b == nil {
		return TradeResult{Summary: "trade failed because one trader was missing"}
	}
	if !hasItems(a, offer.GivingItems) || !hasItems(b, offer.WantingItems) {
		return TradeResult{Summary: "trade failed because someone promised goods they do not have"}
	}

	givingValue := itemsValue(offer.GivingItems)
	wantingValue := itemsValue(offer.WantingItems)
	fairness := tradeFairness(givingValue, wantingValue)
	acceptanceThreshold := float32(0.8)
	acceptanceThreshold -= a.Personality.Agreeableness / 250.0
	acceptanceThreshold -= b.Personality.Agreeableness / 250.0

	haggled := a.Personality.Ambition > 60 || b.Personality.Ambition > 60
	if haggled {
		acceptanceThreshold += 0.08
	}
	if a.Archetype == entities.ArchetypeVillain || b.Archetype == entities.ArchetypeVillain {
		acceptanceThreshold += 0.12
	}

	accepted := fairness >= acceptanceThreshold
	result := TradeResult{
		Accepted:          accepted,
		Haggled:           haggled,
		RelationshipDelta: -6,
		Summary:           "the negotiation stalled and left a bad aftertaste",
	}

	if accepted {
		moveItems(a, b, offer.GivingItems)
		moveItems(b, a, offer.WantingItems)
		result.RelationshipDelta = 8
		result.Summary = "the trade went through and both sides walked away with something useful"
		if givingValue > wantingValue {
			result.AdvantageTo = bID
		} else if wantingValue > givingValue {
			result.AdvantageTo = aID
		}
	} else if a.Archetype == entities.ArchetypeVillain || b.Archetype == entities.ArchetypeVillain {
		result.AdvantageTo = villainParty(a, b)
		result.RelationshipDelta = -12
		result.Summary = "the negotiation turned predatory before it became an exchange"
	}

	applyTradeRelationshipShift(a, b, result.RelationshipDelta)
	event := TradeEvent{
		ID:           fmt.Sprintf("trade_%s_%s_%d", aID, bID, m.world.Calendar.ToMinutes()),
		Day:          m.world.Calendar,
		TraderAID:    aID,
		TraderBID:    bID,
		Offer:        offer,
		Accepted:     accepted,
		Haggled:      haggled,
		DramaScore:   clampWorldInt(25+absInt(givingValue-wantingValue)/2, 0, 100),
		Summary:      result.Summary,
		CurrencyUsed: m.HasCurrency,
	}
	m.TradeHistory = append(m.TradeHistory, event)
	m.world.EventHistory = append(m.world.EventHistory, tradeHistoryEvent(event))
	return result
}

// Inheritance distributes wealth and goods after death.
func Inheritance(deceasedID string, world *World) []InheritanceGrant {
	if world == nil {
		return nil
	}
	deceased := world.GetPoble(deceasedID)
	if deceased == nil {
		if stored, ok := world.pobles[deceasedID]; ok {
			deceased = stored
		}
	}
	if deceased == nil {
		return nil
	}

	heirs := inheritanceHeirs(deceased, world)
	if len(heirs) == 0 {
		return nil
	}

	grants := make([]InheritanceGrant, 0, len(heirs))
	shareMoney := 0
	if deceased.Money > 0 {
		shareMoney = maxInt(1, deceased.Money/len(heirs))
	}
	itemChunks := splitItems(deceased.Inventory, len(heirs))
	revealed := revealInheritanceSecrets(deceased)

	for i, heir := range heirs {
		if heir == nil {
			continue
		}
		heir.Money += shareMoney
		heir.Inventory = append(heir.Inventory, cloneItems(itemChunks[i])...)
		grant := InheritanceGrant{
			RecipientID:     heir.ID,
			Money:           shareMoney,
			Items:           cloneItems(itemChunks[i]),
			RevealedSecrets: append([]string{}, revealed...),
			ConflictRisk:    inheritanceConflictRisk(heir, deceased, world),
		}
		grants = append(grants, grant)
	}

	deceased.Money = maxInt(0, deceased.Money-shareMoney*len(heirs))
	deceased.Inventory = []Item{}
	for _, secretText := range revealed {
		world.EventHistory = append(world.EventHistory, GameEvent{
			ID:           fmt.Sprintf("inheritance_secret_%s_%d", deceased.ID, world.Calendar.ToMinutes()),
			Type:         ai.GameEventBetrayal,
			Time:         world.Calendar,
			PrimaryActor: deceased.ID,
			Participants: append([]string{deceased.ID}, heirIDs(heirs)...),
			Severity:     0.68,
			Valence:      -0.45,
			Description:  secretText,
			Tags:         []string{"inheritance", "secret_revealed"},
		})
	}

	return grants
}

// Theft resolves a theft attempt and its fallout.
func Theft(thiefID string, victimID string, world *World) TheftEvent {
	event := TheftEvent{ThiefID: thiefID, VictimID: victimID}
	if world == nil {
		event.Summary = "the theft could not happen because the world was missing"
		return event
	}
	thief := world.GetPoble(thiefID)
	victim := world.GetPoble(victimID)
	if thief == nil || victim == nil {
		event.Summary = "the theft collapsed because one side was missing"
		return event
	}

	rng := worldRNG(world)
	event.ID = fmt.Sprintf("theft_%s_%s_%d", thiefID, victimID, world.Calendar.ToMinutes())
	event.StolenMoney = minInt(victim.Money, theftMoneyAmount(thief, victim))
	if len(victim.Inventory) > 0 {
		event.StolenItems = cloneItems([]Item{victim.Inventory[0]})
		removeItems(victim, event.StolenItems)
		thief.Inventory = append(thief.Inventory, cloneItems(event.StolenItems)...)
	}
	if event.StolenMoney > 0 {
		victim.Money -= event.StolenMoney
		thief.Money += event.StolenMoney
	}

	discoveryChance := float32(0.28) + victim.Personality.Neuroticism/300.0 - thief.Personality.Openness/500.0
	if thief.Archetype == entities.ArchetypeVillain {
		discoveryChance -= 0.06
	}
	event.Discovered = rng.Float32() < clampWorldFloat(discoveryChance, 0.08, 0.88)
	event.ResentmentCreated = 12
	if event.Discovered {
		event.ResentmentCreated = 26
	}
	applyTheftEmotion(thief, victim, event.ResentmentCreated, event.Discovered)

	if event.Discovered && lawAgainstTheft(world) {
		event.LegalConsequence = propertyLawPenalty(world)
		event.Summary = "the theft was discovered and the settlement demanded a price"
		world.EventHistory = append(world.EventHistory, GameEvent{
			ID:           event.ID,
			Type:         ai.GameEventConflict,
			Time:         world.Calendar,
			PrimaryActor: thiefID,
			TargetID:     victimID,
			Participants: []string{thiefID, victimID},
			Severity:     0.78,
			Valence:      -0.82,
			IsTraumatic:  false,
			Description:  event.Summary,
			Tags:         []string{"theft", "law"},
		})
		return event
	}

	if event.Discovered {
		event.Summary = "the theft was discovered and now both sides know the relationship is poisoned"
	} else {
		event.Summary = "the theft stayed hidden, but the thief still has to live inside the choice"
	}
	world.EventHistory = append(world.EventHistory, GameEvent{
		ID:           event.ID,
		Type:         ai.GameEventSocialNegative,
		Time:         world.Calendar,
		PrimaryActor: thiefID,
		TargetID:     victimID,
		Participants: []string{thiefID, victimID},
		Severity:     0.64,
		Valence:      -0.65,
		Description:  event.Summary,
		Tags:         []string{"theft"},
	})
	return event
}

// Gambling runs a procedural betting session.
func Gambling(participants []string, world *World) GamblingSession {
	session := GamblingSession{Participants: append([]string{}, participants...)}
	if world == nil || len(participants) == 0 {
		session.Summary = "the gambling session never really formed"
		return session
	}

	rng := worldRNG(world)
	session.ID = fmt.Sprintf("gambling_%d", world.Calendar.ToMinutes())
	session.Day = world.Calendar
	session.Outcomes = make([]GamblingOutcome, 0, len(participants))

	active := make([]*Poble, 0, len(participants))
	for _, id := range participants {
		if poble := world.GetPoble(id); poble != nil {
			active = append(active, poble)
		}
	}
	if len(active) == 0 {
		session.Summary = "nobody with money actually showed up to bet"
		return session
	}

	eligibleWinners := make([]string, 0, len(active))
	for i, poble := range active {
		stake := gamblingStakeFor(poble)
		if stake > poble.Money {
			stake = poble.Money
		}
		if stake <= 0 {
			continue
		}
		poble.Money -= stake
		session.Pot += stake
		outcome := GamblingOutcome{PobleID: poble.ID, Stake: stake, Net: -stake}
		session.Outcomes = append(session.Outcomes, outcome)
		eligibleWinners = append(eligibleWinners, poble.ID)
		_ = i
	}
	if len(session.Outcomes) == 0 {
		session.Summary = "the table talked big but no one had enough to stake"
		return session
	}

	winnerID := eligibleWinners[rng.Intn(len(eligibleWinners))]
	session.WinnerID = winnerID
	for i := range session.Outcomes {
		if session.Outcomes[i].PobleID == winnerID {
			session.Outcomes[i].Winnings = session.Pot
			session.Outcomes[i].Net = session.Pot - session.Outcomes[i].Stake
			break
		}
	}
	for _, poble := range active {
		if poble != nil && poble.ID == winnerID {
			poble.Money += session.Pot
			break
		}
	}
	session.DramaScore = clampWorldInt(35+session.Pot/2, 0, 100)
	session.Summary = "the betting table changed who feels lucky and who feels hunted by the day"
	world.EventHistory = append(world.EventHistory, GameEvent{
		ID:           session.ID,
		Type:         ai.GameEventSocialNegative,
		Time:         world.Calendar,
		PrimaryActor: session.WinnerID,
		Participants: append([]string{}, participants...),
		Severity:     0.58,
		Valence:      0.12,
		Description:  session.Summary,
		Tags:         []string{"gambling"},
	})
	return session
}

func runMarketPulse(world *World) *GameEvent {
	if world == nil || world.GetPopulation() < 2 || world.Calendar.Day%3 != 0 {
		return nil
	}
	traders := marketTraders(world)
	if len(traders) < 2 {
		return nil
	}
	a := traders[0]
	b := traders[1]
	seedMarketInventory(a, ResourceFood)
	seedMarketInventory(b, ResourceWood)
	manager := NewEconomyManager(world)
	result := manager.Trade(a.ID, b.ID, TradeoOffer{
		GivingItems:  []Item{{ID: "market_food", Name: "Market Food", Type: "food", Quantity: 1, Value: 3}},
		WantingItems: []Item{{ID: "market_wood", Name: "Market Wood", Type: "wood", Quantity: 1, Value: 4}},
	})
	if len(world.EventHistory) == 0 {
		return nil
	}
	event := world.EventHistory[len(world.EventHistory)-1]
	if result.Accepted && world.TechTree.Unlocked[TechCurrency] {
		a.Money += 1
		b.Money += 1
	}
	return &event
}

func marketTraders(world *World) []*Poble {
	traders := append([]*Poble(nil), world.GetAllPobles()...)
	sort.Slice(traders, func(i, j int) bool {
		left := traders[i].Personality.Ambition + traders[i].Personality.Conscientiousness
		right := traders[j].Personality.Ambition + traders[j].Personality.Conscientiousness
		if left == right {
			return traders[i].ID < traders[j].ID
		}
		return left > right
	})
	return traders
}

func seedMarketInventory(poble *Poble, resource ResourceType) {
	if poble == nil {
		return
	}
	item := marketItem(resource)
	for _, existing := range poble.Inventory {
		if itemKey(existing) == itemKey(item) && existing.Quantity > 0 {
			return
		}
	}
	poble.Inventory = mergeItems(poble.Inventory, []Item{item})
}

func marketItem(resource ResourceType) Item {
	switch resource {
	case ResourceWood:
		return Item{ID: "market_wood", Name: "Market Wood", Type: "wood", Quantity: 1, Value: 4}
	default:
		return Item{ID: "market_food", Name: "Market Food", Type: "food", Quantity: 1, Value: 3}
	}
}

func (m *EconomyManager) syncCurrency() {
	if m == nil || m.world == nil {
		return
	}
	m.HasCurrency = hasCurrency(m.world)
}

func defaultPrices() map[ResourceType]int {
	return map[ResourceType]int{
		ResourceFood:     3,
		ResourceWater:    2,
		ResourceWood:     4,
		ResourceStone:    5,
		ResourceMetal:    8,
		ResourceMedicine: 11,
		ResourceLuxury:   15,
	}
}

func produceResources(island *Island, world *World) map[ResourceType]int {
	produced := map[ResourceType]int{}
	workers := discoveredWorkers(island, world)
	productiveBuildings := 0
	for _, building := range island.Buildings {
		if isProductiveBuilding(building.Type) {
			productiveBuildings++
		}
	}
	share := 0
	remainder := 0
	if productiveBuildings > 0 {
		share = workers / productiveBuildings
		remainder = workers % productiveBuildings
	}
	productiveIndex := 0
	for _, building := range island.Buildings {
		workforce := len(building.Inhabitants)
		if workforce == 0 && workers > 0 && isProductiveBuilding(building.Type) && productiveBuildings > 0 {
			workforce = share
			if productiveIndex < remainder {
				workforce++
			}
			productiveIndex++
		}
		switch building.Type {
		case BuildingFarm:
			produced[ResourceFood] += 3 * workforce
			produced[ResourceWater] += workforce
		case BuildingWorkshop:
			produced[ResourceWood] += workforce
			produced[ResourceStone] += workforce
			if island.Resources[ResourceMetal] > 0 {
				produced[ResourceMetal] += workforce / 2
			}
		case BuildingHospital:
			produced[ResourceMedicine] += maxInt(1, workforce)
		case BuildingTemple:
			produced[ResourceLuxury] += workforce / 2
		case BuildingGovernment:
			produced[ResourceKnowledge] += workforce / 2
		case BuildingHome:
			produced[ResourceWood] += workforce / 3
		}
	}
	if island.Biome == BiomeMystery {
		produced[ResourceLuxury] += 2
	}
	return produced
}

func isProductiveBuilding(buildingType BuildingType) bool {
	switch buildingType {
	case BuildingFarm, BuildingWorkshop, BuildingHospital, BuildingTemple, BuildingGovernment, BuildingHome:
		return true
	default:
		return false
	}
}

func applyScarcityPressure(island *Island, world *World) {
	if island.Resources[ResourceFood] > 0 && island.Resources[ResourceWater] > 0 {
		return
	}
	for _, id := range island.Pobles {
		poble := world.GetPoble(id)
		if poble == nil {
			continue
		}
		if island.Resources[ResourceFood] == 0 {
			poble.Needs.Hunger = clampWorldFloat(poble.Needs.Hunger+20, 0, 100)
		}
		if island.Resources[ResourceWater] == 0 {
			poble.Needs.Thirst = clampWorldFloat(poble.Needs.Thirst+25, 0, 100)
		}
		poble.Mental.Stability = clampWorldInt(poble.Mental.Stability-4, 0, 100)
	}
	if len(island.Pobles) >= 2 {
		world.EventHistory = append(world.EventHistory, GameEvent{
			ID:           fmt.Sprintf("scarcity_%s_%d", island.ID, world.Calendar.ToMinutes()),
			Type:         ai.GameEventConflict,
			Time:         world.Calendar,
			Participants: append([]string{}, island.Pobles...),
			Severity:     0.7,
			Valence:      -0.76,
			Description:  "scarcity turned routine needs into social pressure",
			Tags:         []string{"scarcity", "economy"},
		})
	}
}

func discoveredWorkers(island *Island, world *World) int {
	workers := 0
	for _, id := range island.Pobles {
		if poble := world.GetPoble(id); poble != nil && poble.Needs.Sleep < 70 {
			workers++
		}
	}
	return workers
}

func hasItems(owner *Poble, items []Item) bool {
	counts := inventoryCounts(owner.Inventory)
	for _, item := range items {
		key := itemKey(item)
		if counts[key] < item.Quantity {
			return false
		}
	}
	return true
}

func itemsValue(items []Item) int {
	total := 0
	for _, item := range items {
		value := item.Value
		if value <= 0 {
			value = 1
		}
		total += value * maxInt(1, item.Quantity)
	}
	return total
}

func tradeFairness(givingValue, wantingValue int) float32 {
	if wantingValue <= 0 {
		return 1
	}
	if givingValue >= wantingValue {
		return 1
	}
	return float32(givingValue) / float32(wantingValue)
}

func moveItems(from *Poble, to *Poble, items []Item) {
	removeItems(from, items)
	to.Inventory = mergeItems(to.Inventory, items)
}

func removeItems(owner *Poble, items []Item) {
	for _, removing := range items {
		remaining := removing.Quantity
		for i := range owner.Inventory {
			if remaining <= 0 {
				break
			}
			if itemKey(owner.Inventory[i]) != itemKey(removing) {
				continue
			}
			taken := minInt(owner.Inventory[i].Quantity, remaining)
			owner.Inventory[i].Quantity -= taken
			remaining -= taken
		}
	}

	filtered := owner.Inventory[:0]
	for _, item := range owner.Inventory {
		if item.Quantity > 0 {
			filtered = append(filtered, item)
		}
	}
	owner.Inventory = filtered
}

func mergeItems(existing []Item, items []Item) []Item {
	for _, incoming := range items {
		merged := false
		for i := range existing {
			if itemKey(existing[i]) == itemKey(incoming) {
				existing[i].Quantity += incoming.Quantity
				merged = true
				break
			}
		}
		if !merged {
			existing = append(existing, incoming)
		}
	}
	return existing
}

func inventoryCounts(items []Item) map[string]int {
	counts := map[string]int{}
	for _, item := range items {
		counts[itemKey(item)] += item.Quantity
	}
	return counts
}

func itemKey(item Item) string {
	return item.ID + "|" + item.Name + "|" + item.Type
}

func applyTradeRelationshipShift(a, b *Poble, delta int) {
	ensureWorldRelationship(a, b.ID)
	ensureWorldRelationship(b, a.ID)
	relAB := a.Relationships[b.ID]
	relBA := b.Relationships[a.ID]
	relAB.Trust = clampWorldFloat(relAB.Trust+float32(delta), 0, 100)
	relBA.Trust = clampWorldFloat(relBA.Trust+float32(delta), 0, 100)
	if delta < 0 {
		relAB.Resentment = clampWorldFloat(relAB.Resentment+float32(-delta), 0, 100)
		relBA.Resentment = clampWorldFloat(relBA.Resentment+float32(-delta), 0, 100)
	} else {
		relAB.Affection = clampWorldFloat(relAB.Affection+float32(delta)/2, 0, 100)
		relBA.Affection = clampWorldFloat(relBA.Affection+float32(delta)/2, 0, 100)
	}
	a.Relationships[b.ID] = relAB
	b.Relationships[a.ID] = relBA
}

func ensureWorldRelationship(owner *Poble, targetID string) {
	if owner.Relationships == nil {
		owner.Relationships = map[string]entities.Relationship{}
	}
	if _, ok := owner.Relationships[targetID]; !ok {
		owner.Relationships[targetID] = entities.NewRelationship(targetID, entities.RelationshipAcquaintance)
	}
}

func villainParty(a, b *Poble) string {
	if a.Archetype == entities.ArchetypeVillain {
		return a.ID
	}
	if b.Archetype == entities.ArchetypeVillain {
		return b.ID
	}
	return ""
}

func tradeHistoryEvent(event TradeEvent) GameEvent {
	eventType := ai.GameEventSocialPositive
	valence := float32(0.35)
	if !event.Accepted {
		eventType = ai.GameEventSocialNegative
		valence = -0.35
	}
	return GameEvent{
		ID:           event.ID,
		Type:         eventType,
		Time:         event.Day,
		PrimaryActor: event.TraderAID,
		TargetID:     event.TraderBID,
		Participants: []string{event.TraderAID, event.TraderBID},
		Severity:     float32(event.DramaScore) / 100,
		Valence:      valence,
		Description:  event.Summary,
		Tags:         []string{"trade"},
	}
}

func inheritanceHeirs(deceased *Poble, world *World) []*Poble {
	heirs := []*Poble{}
	seen := map[string]bool{}
	for _, childID := range deceased.Children {
		if child := world.GetPoble(childID); child != nil && !seen[child.ID] {
			heirs = append(heirs, child)
			seen[child.ID] = true
		}
	}
	for id, rel := range deceased.Relationships {
		if rel.Type == entities.RelationshipSpouse || rel.Type == entities.RelationshipLover || rel.Type == entities.RelationshipFamily {
			if heir := world.GetPoble(id); heir != nil && !seen[heir.ID] {
				heirs = append(heirs, heir)
				seen[heir.ID] = true
			}
		}
	}
	return heirs
}

func splitItems(items []Item, parts int) [][]Item {
	chunks := make([][]Item, parts)
	if parts <= 0 {
		return chunks
	}
	for i, item := range items {
		target := i % parts
		chunks[target] = append(chunks[target], item)
	}
	return chunks
}

func revealInheritanceSecrets(deceased *Poble) []string {
	revealed := []string{}
	if deceased.Money >= 100 {
		revealed = append(revealed, "hidden money surfaced during inheritance and changed what the family thought it knew")
	}
	for _, item := range deceased.Inventory {
		if hasTag(item.Tags, "hidden_property") || strings.Contains(strings.ToLower(item.Name), "hidden") {
			revealed = append(revealed, "an unknown property or stash was revealed by the dead person's things")
			break
		}
	}
	for _, secret := range deceased.Secrets {
		if secret.Type == entities.SecretChild || secret.Type == entities.SecretCriminalAct {
			revealed = append(revealed, "inheritance forced one of the dead person's secrets into the open")
			break
		}
	}
	return uniqueStringsLocal(revealed)
}

func inheritanceConflictRisk(heir *Poble, deceased *Poble, world *World) int {
	risk := 35
	if heir != nil {
		risk += int(heir.Personality.Ambition*0.18 + heir.Personality.Jealousy*0.16)
	}
	if len(revealInheritanceSecrets(deceased)) > 0 {
		risk += 20
	}
	if world != nil && len(inheritanceHeirs(deceased, world)) > 2 {
		risk += 10
	}
	return clampWorldInt(risk, 0, 100)
}

func heirIDs(heirs []*Poble) []string {
	ids := make([]string, 0, len(heirs))
	for _, heir := range heirs {
		if heir != nil {
			ids = append(ids, heir.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func theftMoneyAmount(thief *Poble, victim *Poble) int {
	amount := maxInt(1, victim.Money/4)
	if thief.Archetype == entities.ArchetypeVillain {
		amount = maxInt(1, victim.Money/2)
	}
	return amount
}

func applyTheftEmotion(thief *Poble, victim *Poble, resentment int, discovered bool) {
	ensureWorldRelationship(thief, victim.ID)
	ensureWorldRelationship(victim, thief.ID)
	thiefRel := thief.Relationships[victim.ID]
	victimRel := victim.Relationships[thief.ID]

	thiefRel.Resentment = clampWorldFloat(thiefRel.Resentment+float32(resentment/2), 0, 100)
	thief.Mental.Stability = clampWorldInt(thief.Mental.Stability-3, 0, 100)
	if discovered {
		victimRel.Resentment = clampWorldFloat(victimRel.Resentment+float32(resentment), 0, 100)
		victimRel.Trust = clampWorldFloat(victimRel.Trust-float32(resentment), 0, 100)
	} else {
		thiefRel.Trust = clampWorldFloat(thiefRel.Trust-6, 0, 100)
	}
	thief.Relationships[victim.ID] = thiefRel
	victim.Relationships[thief.ID] = victimRel
}

func lawAgainstTheft(world *World) bool {
	if world == nil || world.Government == nil {
		return false
	}
	for _, law := range world.Government.Laws {
		if law.ID == "law_property" && law.IsEnforced {
			return true
		}
	}
	return false
}

func propertyLawPenalty(world *World) string {
	if world == nil || world.Government == nil {
		return ""
	}
	for _, law := range world.Government.Laws {
		if law.ID == "law_property" {
			return law.Penalty
		}
	}
	return ""
}

func gamblingStakeFor(poble *Poble) int {
	if poble == nil || poble.Money <= 0 {
		return 0
	}
	stake := maxInt(1, poble.Money/10)
	if poble.Archetype == entities.ArchetypeAddict {
		stake = maxInt(stake, poble.Money/3)
	}
	if poble.Personality.Ambition > 70 {
		stake += maxInt(1, poble.Money/12)
	}
	return minInt(stake, poble.Money)
}

func worldRNG(world *World) *rand.Rand {
	if world.rng == nil {
		world.rng = rand.New(rand.NewSource(int64(world.Calendar.ToMinutes() + 17)))
	}
	return world.rng
}

func cloneItems(items []Item) []Item {
	cloned := make([]Item, len(items))
	copy(cloned, items)
	return cloned
}

func hasTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

func uniqueStringsLocal(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func clampWorldInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
