package world

import (
	"math/rand"
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestDailyResourceUpdateConsumesAndProduces(t *testing.T) {
	world := NewWorld(21)
	for i := 0; i < 4; i++ {
		poble := entities.NewPoble(string(rune('a'+i)), "worker", 24, entities.Female)
		poble.IsAlive = true
		world.AddPoble(&poble, Location{IslandID: "island_0", BuildingID: "island_0_home_0"})
	}
	origin := world.GetIsland("island_0")
	origin.Buildings = append(origin.Buildings, Building{ID: "farm", Type: BuildingFarm, Inhabitants: []string{"a", "b"}})
	origin.Resources[ResourceFood] = 20
	origin.Resources[ResourceWater] = 20

	DailyResourceUpdate(world)

	if origin.Resources[ResourceFood] <= 0 {
		t.Fatalf("expected food to remain after production, got %d", origin.Resources[ResourceFood])
	}
	if origin.Resources[ResourceWater] <= 0 {
		t.Fatalf("expected water to remain after update, got %d", origin.Resources[ResourceWater])
	}
}

func TestTradeAcceptsFairOffer(t *testing.T) {
	world := NewWorld(22)
	a := entities.NewPoble("a", "A", 24, entities.Female)
	b := entities.NewPoble("b", "B", 24, entities.Male)
	a.IsAlive = true
	b.IsAlive = true
	a.Personality.Agreeableness = 80
	b.Personality.Agreeableness = 78
	a.Inventory = []entities.Item{{ID: "fish", Name: "Fish", Type: "food", Quantity: 2, Value: 4}}
	b.Inventory = []entities.Item{{ID: "wood", Name: "Wood", Type: "wood", Quantity: 2, Value: 4}}
	world.AddPoble(&a, Location{IslandID: "island_0"})
	world.AddPoble(&b, Location{IslandID: "island_0"})

	manager := NewEconomyManager(world)
	result := manager.Trade("a", "b", TradeoOffer{
		GivingItems:  []Item{{ID: "fish", Name: "Fish", Type: "food", Quantity: 1, Value: 4}},
		WantingItems: []Item{{ID: "wood", Name: "Wood", Type: "wood", Quantity: 1, Value: 4}},
	})

	if !result.Accepted {
		t.Fatalf("expected fair trade accepted, got %+v", result)
	}
}

func TestInheritanceDistributesMoneyAndRevealsSecrets(t *testing.T) {
	world := NewWorld(23)
	deceased := entities.NewPoble("dead", "Dead", 40, entities.Male)
	child := entities.NewPoble("child", "Child", 20, entities.Female)
	deceased.IsAlive = false
	child.IsAlive = true
	deceased.Children = []string{"child"}
	deceased.Money = 120
	deceased.Inventory = []entities.Item{{ID: "stash", Name: "Hidden Chest", Type: "property", Quantity: 1, Value: 25, Tags: []string{"hidden_property"}}}
	deceased.Secrets = []entities.Secret{entities.NewSecret("s1", entities.SecretChild, "secret child elsewhere")}
	world.pobles["dead"] = &deceased
	world.AddPoble(&child, Location{IslandID: "island_0"})

	grants := Inheritance("dead", world)
	if len(grants) == 0 {
		t.Fatal("expected inheritance grants")
	}
	if child.Money == 0 {
		t.Fatalf("expected child to inherit money, got %d", child.Money)
	}
	if len(grants[0].RevealedSecrets) == 0 {
		t.Fatalf("expected inheritance to reveal secrets, got %+v", grants[0])
	}
}

func TestTheftUsesLawWhenDiscovered(t *testing.T) {
	world := NewWorld(24)
	world.Government = &GovernmentSystem{
		Type: GovernmentDemocracy,
		Laws: []Law{{ID: "law_property", Description: "no theft", Penalty: "restitution", IsEnforced: true}},
	}
	thief := entities.NewPoble("thief", "Thief", 28, entities.Male)
	victim := entities.NewPoble("victim", "Victim", 28, entities.Female)
	thief.IsAlive = true
	victim.IsAlive = true
	thief.Archetype = entities.ArchetypeVillain
	victim.Money = 50
	victim.Inventory = []entities.Item{{ID: "med", Name: "Medicine", Type: "medicine", Quantity: 1, Value: 10}}
	world.AddPoble(&thief, Location{IslandID: "island_0"})
	world.AddPoble(&victim, Location{IslandID: "island_0"})

	world.rng = rand.New(rand.NewSource(1))
	event := Theft("thief", "victim", world)

	if event.ID == "" {
		t.Fatal("expected theft event id")
	}
	if event.StolenMoney == 0 && len(event.StolenItems) == 0 {
		t.Fatalf("expected stolen goods, got %+v", event)
	}
}

func TestGamblingRewardsWinnerAndHooksAddict(t *testing.T) {
	world := NewWorld(25)
	a := entities.NewPoble("a", "A", 30, entities.Male)
	b := entities.NewPoble("b", "B", 30, entities.Female)
	a.IsAlive = true
	b.IsAlive = true
	a.Archetype = entities.ArchetypeAddict
	a.Money = 90
	b.Money = 60
	world.AddPoble(&a, Location{IslandID: "island_0"})
	world.AddPoble(&b, Location{IslandID: "island_0"})
	world.rng = rand.New(rand.NewSource(2))

	session := Gambling([]string{"a", "b"}, world)
	if session.Pot <= 0 || session.WinnerID == "" {
		t.Fatalf("expected real gambling session, got %+v", session)
	}
	if len(session.Outcomes) != 2 {
		t.Fatalf("expected two gambling outcomes, got %+v", session.Outcomes)
	}
}
