package character

import (
	"fmt"
	"log/slog"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/d2go/pkg/data/skill"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/d2go/pkg/data/state"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/utils"
)

const (
	druMaxAttacksLoop = 20
	druMinDistance    = 2
	druMaxDistance    = 8
)

type WindDruid struct {
	CharacterBuild
	*game.HID
}

func (s WindDruid) CheckKeyBindings() []skill.ID {
	requireKeybindings := []skill.ID{skill.Hurricane, skill.OakSage, skill.CycloneArmor, skill.TomeOfTownPortal}
	missingKeybindings := []skill.ID{}

	for _, cskill := range requireKeybindings {
		if _, found := s.Data.KeyBindings.KeyBindingForSkill(cskill); !found {
			missingKeybindings = append(missingKeybindings, cskill)
		}
	}

	if len(missingKeybindings) > 0 {
		s.Logger.Debug("There are missing required key bindings.", slog.Any("Bindings", missingKeybindings))
	}

	return missingKeybindings
}

func (s WindDruid) KillMonsterSequence(
	monsterSelector func(d game.Data) (data.UnitID, bool),
	skipOnImmunities []stat.Resist,
) error {
	completedAttackLoops := 0
	previousUnitID := 0

	for {
		id, found := monsterSelector(*s.Data)
		if !found {
			return nil
		}
		if previousUnitID != int(id) {
			completedAttackLoops = 0
		}

		if !s.preBattleChecks(id, skipOnImmunities) {
			return nil
		}

		s.RecastBuffs()

		if completedAttackLoops >= maxAttacksLoop {
			return nil
		}

		monster, found := s.Data.Monsters.FindByID(id)
		if !found {
			s.Logger.Info("Monster not found", slog.String("monster", fmt.Sprintf("%v", monster)))
			return nil
		}

		// Add a random movement, maybe tornado is not hitting the target
		if previousUnitID == int(id) {
			if monster.Stats[stat.Life] > 0 {
				s.PathFinder.RandomMovement()
			}
			return nil
		}

		step.PrimaryAttack(
			id,
			3,
			true,
			step.Distance(druMinDistance, druMaxDistance),
		)

		completedAttackLoops++
		previousUnitID = int(id)
	}
}

func (s WindDruid) killMonster(npc npc.ID, t data.MonsterType) error {
	return s.KillMonsterSequence(func(d game.Data) (data.UnitID, bool) {
		m, found := d.Monsters.FindOne(npc, t)
		if !found {
			return 0, false
		}

		return m.UnitID, true
	}, nil)
}

func (s WindDruid) RecastBuffs() {
	skills := []skill.ID{skill.Hurricane, skill.OakSage, skill.CycloneArmor}
	states := []state.State{state.Hurricane, state.Oaksage, state.Cyclonearmor}

	for i, druSkill := range skills {
		if kb, found := s.Data.KeyBindings.KeyBindingForSkill(druSkill); found {
			if !s.Data.PlayerUnit.States.HasState(states[i]) {
				s.HID.PressKeyBinding(kb)
				utils.Sleep(180)
				s.HID.Click(game.RightButton, 640, 340)
				utils.Sleep(100)
			}
		}
	}
}

func (s WindDruid) BuffSkills() (buffs []skill.ID) {
	if _, found := s.Data.KeyBindings.KeyBindingForSkill(skill.CycloneArmor); found {
		buffs = append(buffs, skill.CycloneArmor)
	}
	if _, ravenFound := s.Data.KeyBindings.KeyBindingForSkill(skill.Raven); ravenFound {
		buffs = append(buffs, skill.Raven, skill.Raven, skill.Raven, skill.Raven, skill.Raven)
	}
	if _, hurricaneFound := s.Data.KeyBindings.KeyBindingForSkill(skill.Hurricane); hurricaneFound {
		buffs = append(buffs, skill.Hurricane)
	}
	return buffs
}

func (s WindDruid) PreCTABuffSkills() (skills []skill.ID) {
	needsBear := true
	wolves := 5
	direWolves := 3
	needsOak := true
	needsCreeper := true

	for _, monster := range s.Data.Monsters {
		if monster.IsPet() {
			if monster.Name == npc.DruBear {
				needsBear = false
			}
			if monster.Name == npc.DruFenris {
				direWolves--
			}
			if monster.Name == npc.DruSpiritWolf {
				wolves--
			}
			if monster.Name == npc.OakSage {
				needsOak = false
			}
			if monster.Name == npc.DruCycleOfLife {
				needsCreeper = false
			}
			if monster.Name == npc.VineCreature {
				needsCreeper = false
			}
			if monster.Name == npc.DruPlaguePoppy {
				needsCreeper = false
			}
		}
	}

	if s.Data.PlayerUnit.States.HasState(state.Oaksage) {
		needsOak = false
	}

	_, foundDireWolf := s.Data.KeyBindings.KeyBindingForSkill(skill.SummonDireWolf)
	_, foundWolf := s.Data.KeyBindings.KeyBindingForSkill(skill.SummonSpiritWolf)
	_, foundBear := s.Data.KeyBindings.KeyBindingForSkill(skill.SummonGrizzly)
	_, foundOak := s.Data.KeyBindings.KeyBindingForSkill(skill.OakSage)
	_, foundSolarCreeper := s.Data.KeyBindings.KeyBindingForSkill(skill.SolarCreeper)
	_, foundCarrionCreeper := s.Data.KeyBindings.KeyBindingForSkill(skill.CarrionVine)
	_, foundPoisonCreeper := s.Data.KeyBindings.KeyBindingForSkill(skill.PoisonCreeper)

	if foundWolf {
		for i := 0; i < wolves; i++ {
			skills = append(skills, skill.SummonSpiritWolf)
		}
	}
	if foundDireWolf {
		for i := 0; i < direWolves; i++ {
			skills = append(skills, skill.SummonDireWolf)
		}
	}
	if foundBear && needsBear {
		skills = append(skills, skill.SummonGrizzly)
	}
	if foundOak && needsOak {
		skills = append(skills, skill.OakSage)
	}
	if foundSolarCreeper && needsCreeper {
		skills = append(skills, skill.SolarCreeper)
	}
	if foundCarrionCreeper && needsCreeper {
		skills = append(skills, skill.CarrionVine)
	}
	if foundPoisonCreeper && needsCreeper {
		skills = append(skills, skill.PoisonCreeper)
	}

	return skills
}
