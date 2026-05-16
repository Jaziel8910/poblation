package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/world"
)

type houseFocus string

const (
	houseFocusObjects houseFocus = "objects"
	houseFocusContent houseFocus = "content"
)

type houseObjectKind string

const (
	houseObjectDiary  houseObjectKind = "diary"
	houseObjectLetter houseObjectKind = "letter"
	houseObjectBox    houseObjectKind = "box"
	houseObjectMoney  houseObjectKind = "money"
	houseObjectKeys   houseObjectKind = "keys"
	houseObjectPhoto  houseObjectKind = "photo"
	houseObjectItem   houseObjectKind = "item"
)

type houseObject struct {
	Kind        houseObjectKind
	Icon        string
	Title       string
	Summary     string
	Item        *entities.Item
	Interactive bool
}

// HouseModel renders the inside of a private house.
type HouseModel struct {
	HouseOwner     *entities.Poble
	Building       *world.Building
	CurrentRoom    string
	state          AppStateSnapshot
	leftViewport   viewport.Model
	rightViewport  viewport.Model
	objects        []houseObject
	selectedObject int
	focus          houseFocus
	pendingBoxOpen bool
}

var (
	housePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Background(backgroundColor).
			Foreground(primaryColor).
			Padding(0, 1)

	houseHeaderStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)
)

// NewHouseModel creates the house exploration screen.
func NewHouseModel() HouseModel {
	return HouseModel{
		CurrentRoom:   "Sala principal",
		leftViewport:  viewport.New(28, 12),
		rightViewport: viewport.New(36, 12),
		focus:         houseFocusObjects,
	}
}

// Init satisfies tea.Model.
func (m HouseModel) Init() tea.Cmd {
	return nil
}

// SyncAppState reloads house data from the shared app snapshot.
func (m HouseModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.resolveHouse()
	m.resizeViewports()
	m.syncViewports()
	return m
}

func (m HouseModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resizeViewports()
	m.syncViewports()
	return m
}

// OnEnter resets transient confirmation state when opening a house.
func (m HouseModel) OnEnter() (tea.Model, tea.Cmd) {
	m.pendingBoxOpen = false
	m.focus = houseFocusObjects
	m.selectedObject = clampInt(m.selectedObject, 0, maxInt(0, len(m.objects)-1))
	m.syncViewports()
	return m, nil
}

// Update handles house object selection and scroll.
func (m HouseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case tea.KeyMsg:
		return m.handleKey(typed)
	default:
		return m, nil
	}
}

// View renders the two-panel house layout.
func (m HouseModel) View() string {
	leftWidth, rightWidth := m.panelWidths()
	left := housePanelStyle.Width(leftWidth).Height(maxInt(10, m.state.Height-1)).Render(m.leftViewport.View())
	right := housePanelStyle.Width(rightWidth).Height(maxInt(10, m.state.Height-1)).Render(m.rightViewport.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m HouseModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.focus == houseFocusObjects {
			m.focus = houseFocusContent
		} else {
			m.focus = houseFocusObjects
		}
	case "up":
		if m.focus == houseFocusObjects {
			m.selectedObject--
			m.selectedObject = clampInt(m.selectedObject, 0, maxInt(0, len(m.objects)-1))
			m.pendingBoxOpen = false
		} else {
			m.rightViewport.ScrollUp(1)
		}
	case "down":
		if m.focus == houseFocusObjects {
			m.selectedObject++
			m.selectedObject = clampInt(m.selectedObject, 0, maxInt(0, len(m.objects)-1))
			m.pendingBoxOpen = false
		} else {
			m.rightViewport.ScrollDown(1)
		}
	case "pgup":
		if m.focus == houseFocusObjects {
			m.leftViewport.ScrollUp(maxInt(1, m.leftViewport.Height/2))
		} else {
			m.rightViewport.ScrollUp(maxInt(1, m.rightViewport.Height/2))
		}
	case "pgdown":
		if m.focus == houseFocusObjects {
			m.leftViewport.ScrollDown(maxInt(1, m.leftViewport.Height/2))
		} else {
			m.rightViewport.ScrollDown(maxInt(1, m.rightViewport.Height/2))
		}
	case "enter":
		m.handleEnter()
	}
	m.syncViewports()
	return m, nil
}

func (m *HouseModel) handleEnter() {
	selected := m.currentObject()
	if selected == nil || selected.Kind != houseObjectBox {
		return
	}
	if !m.pendingBoxOpen {
		m.pendingBoxOpen = true
		return
	}
	m.pendingBoxOpen = false
}

func (m *HouseModel) resolveHouse() {
	m.HouseOwner = nil
	m.Building = nil
	if m.state.World == nil {
		m.objects = nil
		return
	}

	building, owner := resolveHouseSelection(m.state.World, m.state.SelectedBuildingID, m.state.SelectedPobleID)
	m.Building = building
	m.HouseOwner = owner
	m.CurrentRoom = houseRoomLabel(building)
	m.objects = buildHouseObjects(building, owner)
	m.selectedObject = clampInt(m.selectedObject, 0, maxInt(0, len(m.objects)-1))
}

func (m *HouseModel) resizeViewports() {
	leftWidth, rightWidth := m.panelWidths()
	innerHeight := maxInt(8, m.state.Height-4)
	m.leftViewport.Width = maxInt(20, leftWidth-housePanelStyle.GetHorizontalFrameSize())
	m.leftViewport.Height = innerHeight
	m.rightViewport.Width = maxInt(24, rightWidth-housePanelStyle.GetHorizontalFrameSize())
	m.rightViewport.Height = innerHeight
}

func (m *HouseModel) syncViewports() {
	m.leftViewport.SetContent(m.renderObjectList())
	m.rightViewport.SetContent(m.renderObjectContent())
}

func (m HouseModel) renderObjectList() string {
	header := "OBJETOS EN LA CASA"
	if m.Building != nil {
		header = header + "\n" + mutedStyle.Render(fmt.Sprintf("%s · %s", m.Building.Name, m.CurrentRoom))
	}
	if len(m.objects) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, houseHeaderStyle.Render(header), "", mutedStyle.Render("No hay nada que revisar aqui."))
	}

	lines := []string{houseHeaderStyle.Render(header), ""}
	for index, object := range m.objects {
		line := fmt.Sprintf("%s %s", object.Icon, object.Title)
		if index == m.selectedObject {
			prefix := ">"
			if m.focus != houseFocusObjects {
				prefix = "•"
			}
			lines = append(lines, exploreSelectedStyle.Render(prefix+" "+line))
			if strings.TrimSpace(object.Summary) != "" {
				lines = append(lines, mutedStyle.Render("  "+truncateRunes(object.Summary, maxInt(18, m.leftViewport.Width-4))))
			}
			continue
		}
		lines = append(lines, "  "+line)
	}
	lines = append(lines, "", mutedStyle.Render("TAB cambia foco · ENTER abre la caja"))
	return strings.Join(lines, "\n")
}

func (m HouseModel) renderObjectContent() string {
	header := "CONTENIDO"
	if m.HouseOwner != nil {
		header = fmt.Sprintf("%s\n%s", header, mutedStyle.Render("Casa de "+m.HouseOwner.Name))
	}
	body := mutedStyle.Render("Selecciona un objeto.")
	if object := m.currentObject(); object != nil {
		body = m.describeObject(*object)
	}
	return lipgloss.JoinVertical(lipgloss.Left, houseHeaderStyle.Render(header), "", body)
}

func (m HouseModel) currentObject() *houseObject {
	if len(m.objects) == 0 {
		return nil
	}
	index := clampInt(m.selectedObject, 0, len(m.objects)-1)
	return &m.objects[index]
}

func (m HouseModel) describeObject(object houseObject) string {
	switch object.Kind {
	case houseObjectDiary:
		return renderHouseMarkdown(m.diaryMarkdown())
	case houseObjectLetter:
		return renderHouseMarkdown(m.letterMarkdown())
	case houseObjectPhoto:
		return renderHouseMarkdown(m.photoMarkdown())
	case houseObjectMoney:
		return renderHouseMarkdown(m.moneyMarkdown())
	case houseObjectKeys:
		return renderHouseMarkdown(m.keysMarkdown())
	case houseObjectBox:
		return renderHouseMarkdown(m.boxMarkdown())
	case houseObjectItem:
		return renderHouseMarkdown(m.itemMarkdown(object))
	default:
		return mutedStyle.Render("Nada especial.")
	}
}

func (m HouseModel) diaryMarkdown() string {
	lines := []string{"## Diario"}
	if m.Building != nil && len(m.Building.PrivateDiaryEntries) > 0 {
		for _, entry := range m.Building.PrivateDiaryEntries {
			lines = append(lines, fmt.Sprintf("### Dia %d · %02d:%02d", entry.Day.Day, entry.Day.Hour, entry.Day.Minute))
			lines = append(lines, entry.Content)
		}
		return strings.Join(lines, "\n\n")
	}
	if m.HouseOwner != nil && len(m.HouseOwner.Memories) > 0 {
		for index, memory := range m.HouseOwner.Memories {
			if index >= 4 {
				break
			}
			lines = append(lines, fmt.Sprintf("### Dia %d", memory.Timestamp.Day))
			lines = append(lines, memory.Summary)
		}
		return strings.Join(lines, "\n\n")
	}
	lines = append(lines, "No hay paginas abiertas. Solo la sensacion de que alguien escribio aqui para no explotar.")
	return strings.Join(lines, "\n\n")
}

func (m HouseModel) letterMarkdown() string {
	name := "alguien"
	if m.HouseOwner != nil {
		name = m.HouseOwner.Name
	}
	lines := []string{"## Carta sin enviar"}
	if m.HouseOwner != nil {
		if secret := firstHiddenSecret(m.HouseOwner); secret != nil {
			lines = append(lines,
				fmt.Sprintf("**Para:** quien sea que pudiera soportar escuchar a %s", name),
				"",
				secret.Content,
			)
			return strings.Join(lines, "\n")
		}
		if memory := latestMemory(m.HouseOwner); memory != nil {
			lines = append(lines,
				fmt.Sprintf("**Para:** alguien que todavia no sabe que %s piensa esto", name),
				"",
				"Si alguna vez lees esto, no era valentia. Era cansancio.",
				"",
				memory.Summary,
			)
			return strings.Join(lines, "\n")
		}
	}
	lines = append(lines, "No tiene destinatario claro. Solo frases rotas y una despedida que nunca se atrevio a terminar.")
	return strings.Join(lines, "\n")
}

func (m HouseModel) photoMarkdown() string {
	lines := []string{"## Foto"}
	if m.HouseOwner != nil {
		if memory := latestMemory(m.HouseOwner); memory != nil {
			lines = append(lines, "La imagen parece hecha para sostener un recuerdo a la fuerza.", "", memory.Summary)
			return strings.Join(lines, "\n")
		}
		if len(m.HouseOwner.Children) > 0 {
			lines = append(lines, "Una foto con esquinas gastadas. Hay una ternura rara ahi, como si el tiempo hubiera pedido disculpas muy tarde.")
			return strings.Join(lines, "\n")
		}
	}
	lines = append(lines, "Una imagen quieta. Rostros medio borrados. Algo paso aqui y nadie quiso tirarla.")
	return strings.Join(lines, "\n")
}

func (m HouseModel) moneyMarkdown() string {
	amount := 0
	if m.HouseOwner != nil {
		amount = m.HouseOwner.Money
	}
	return fmt.Sprintf("## Dinero\n\nHay **%d** monedas guardadas aqui. No suficiente para paz, pero suficiente para pelea.", amount)
}

func (m HouseModel) keysMarkdown() string {
	lines := []string{
		"## Llaves extranas",
		"",
		"Metal frio, forma rara, ningun llavero amable.",
		"",
		"No abren nada aqui mismo. Igual cuesta soltarlas. Se sienten como promesa o amenaza, y a veces esas dos cosas son lo mismo.",
	}
	return strings.Join(lines, "\n")
}

func (m HouseModel) boxMarkdown() string {
	if !m.pendingBoxOpen {
		lines := []string{
			"## Caja cerrada",
			"",
			"Esta cerrada. No parece imposible de abrir, solo personal.",
			"",
			"Presiona **ENTER** otra vez para abrirla.",
		}
		if m.state.IsDirectorMode {
			lines = append(lines, "", "_Modo Director: podrias dejar un rastro visible._")
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{"## Caja abierta", ""}
	if m.HouseOwner != nil {
		if secret := firstHiddenSecret(m.HouseOwner); secret != nil {
			lines = append(lines, "**Dentro:**", "", "- Un papel doblado con algo que no queria decir en voz alta.", "- "+secret.Content)
			return strings.Join(lines, "\n")
		}
	}
	if m.Building != nil && len(m.Building.Inventory) > 0 {
		lines = append(lines, "**Dentro:**")
		for _, item := range m.Building.Inventory {
			lines = append(lines, fmt.Sprintf("- %s x%d", item.Name, item.Quantity))
		}
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "Adentro hay menos tesoro que tension: tela vieja, polvo y la idea de que alguien una vez escondio algo aqui.")
	return strings.Join(lines, "\n")
}

func (m HouseModel) itemMarkdown(object houseObject) string {
	if object.Item == nil {
		return "## Objeto\n\nTiene peso, textura y muy poca historia visible."
	}
	item := object.Item
	lines := []string{
		"## " + item.Name,
		"",
		fmt.Sprintf("Cantidad: **%d**", item.Quantity),
		fmt.Sprintf("Tipo: **%s**", strings.ToLower(item.Type)),
	}
	if item.Value > 0 {
		lines = append(lines, fmt.Sprintf("Valor: **%d**", item.Value))
	}
	if len(item.Tags) > 0 {
		lines = append(lines, "", "Tags: "+strings.Join(item.Tags, ", "))
	}
	lines = append(lines, "", physicalDescriptionForItem(*item))
	return strings.Join(lines, "\n")
}

func (m HouseModel) panelWidths() (int, int) {
	totalWidth := maxInt(58, m.state.Width)
	leftWidth := int(float64(totalWidth) * 0.38)
	if leftWidth < 24 {
		leftWidth = 24
	}
	if leftWidth > totalWidth-28 {
		leftWidth = totalWidth - 28
	}
	return leftWidth, totalWidth - leftWidth
}

func buildHouseObjects(building *world.Building, owner *entities.Poble) []houseObject {
	if building == nil && owner == nil {
		return nil
	}

	objects := []houseObject{}
	if building != nil && (building.HasPrivateDiary || len(building.PrivateDiaryEntries) > 0 || (owner != nil && len(owner.Memories) > 0)) {
		objects = append(objects, houseObject{Kind: houseObjectDiary, Icon: "📓", Title: "Diario", Summary: "Paginas privadas y cosas que nunca salieron por la boca.", Interactive: true})
	}
	if owner != nil && (len(owner.Secrets) > 0 || len(owner.Memories) > 0) {
		objects = append(objects, houseObject{Kind: houseObjectLetter, Icon: "💌", Title: "Carta sin enviar", Summary: "Una verdad que se quedo doblada.", Interactive: true})
	}
	objects = append(objects, houseObject{Kind: houseObjectBox, Icon: "📦", Title: "Caja cerrada", Summary: "Necesita confirmacion para abrir.", Interactive: true})
	if owner != nil && owner.Money > 0 {
		objects = append(objects, houseObject{Kind: houseObjectMoney, Icon: "💰", Title: "Dinero", Summary: fmt.Sprintf("%d monedas guardadas.", owner.Money)})
	}
	objects = append(objects, houseObject{Kind: houseObjectKeys, Icon: "🔑", Title: "Llaves extrañas", Summary: "Prometen mas de lo que explican."})
	if owner != nil && (len(owner.Memories) > 0 || len(owner.Children) > 0 || owner.HomeID != "") {
		objects = append(objects, houseObject{Kind: houseObjectPhoto, Icon: "🖼", Title: "Fotos", Summary: "Rostros, pruebas y cosas que siguen vivas en papel."})
	}
	if building != nil {
		for index := range building.Inventory {
			item := building.Inventory[index]
			objects = append(objects, houseObject{
				Kind:    houseObjectItem,
				Icon:    iconForItem(item),
				Title:   item.Name,
				Summary: fmt.Sprintf("%s x%d", strings.ToLower(item.Type), item.Quantity),
				Item:    &building.Inventory[index],
			})
		}
	}
	if owner != nil {
		for index := range owner.Inventory {
			item := owner.Inventory[index]
			objects = append(objects, houseObject{
				Kind:    houseObjectItem,
				Icon:    iconForItem(item),
				Title:   item.Name,
				Summary: fmt.Sprintf("del dueño · %s x%d", strings.ToLower(item.Type), item.Quantity),
				Item:    &owner.Inventory[index],
			})
		}
	}
	return objects
}

func resolveHouseSelection(w *world.World, buildingID, ownerID string) (*world.Building, *entities.Poble) {
	if w == nil {
		return nil, nil
	}
	if building, _, ok := findBuildingByID(w, buildingID); ok {
		buildingCopy := building
		owner := resolveHouseOwner(w, &buildingCopy, ownerID)
		return &buildingCopy, owner
	}
	if ownerID != "" {
		if owner := w.GetPoble(ownerID); owner != nil && owner.HomeID != "" {
			if building, _, ok := findBuildingByID(w, owner.HomeID); ok {
				buildingCopy := building
				return &buildingCopy, owner
			}
		}
	}
	if owner := w.GetPoble(ownerID); owner != nil {
		return fallbackHouseForOwner(w, owner)
	}
	if owner := w.GetPoble(buildingID); owner != nil {
		return fallbackHouseForOwner(w, owner)
	}
	for _, island := range w.Islands {
		if island == nil {
			continue
		}
		for index := range island.Buildings {
			if island.Buildings[index].Type == world.BuildingHome {
				buildingCopy := island.Buildings[index]
				owner := resolveHouseOwner(w, &buildingCopy, "")
				return &buildingCopy, owner
			}
		}
	}
	return nil, nil
}

func fallbackHouseForOwner(w *world.World, owner *entities.Poble) (*world.Building, *entities.Poble) {
	if w == nil || owner == nil {
		return nil, nil
	}
	if owner.HomeID != "" {
		if building, _, ok := findBuildingByID(w, owner.HomeID); ok {
			buildingCopy := building
			return &buildingCopy, owner
		}
	}
	for _, island := range w.Islands {
		if island == nil {
			continue
		}
		for index := range island.Buildings {
			building := island.Buildings[index]
			if building.OwnerID == owner.ID {
				buildingCopy := building
				return &buildingCopy, owner
			}
			for _, inhabitant := range building.Inhabitants {
				if inhabitant == owner.ID {
					buildingCopy := building
					return &buildingCopy, owner
				}
			}
		}
	}
	return nil, owner
}

func resolveHouseOwner(w *world.World, building *world.Building, ownerID string) *entities.Poble {
	if w == nil || building == nil {
		return nil
	}
	if ownerID != "" {
		if owner := w.GetPoble(ownerID); owner != nil {
			return owner
		}
	}
	if building.OwnerID != "" {
		if owner := w.GetPoble(building.OwnerID); owner != nil {
			return owner
		}
	}
	for _, inhabitant := range building.Inhabitants {
		if owner := w.GetPoble(inhabitant); owner != nil {
			return owner
		}
	}
	return nil
}

func houseRoomLabel(building *world.Building) string {
	if building == nil {
		return "Interior"
	}
	switch building.Type {
	case world.BuildingHome:
		return "Sala principal"
	case world.BuildingWorkshop:
		return "Mesa de trabajo"
	case world.BuildingTemple:
		return "Sala silenciosa"
	case world.BuildingHospital:
		return "Cuarto de curas"
	default:
		return "Cuarto comun"
	}
}

func renderHouseMarkdown(content string) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(72),
	)
	if err != nil {
		return content
	}
	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return rendered
}

func firstHiddenSecret(owner *entities.Poble) *entities.Secret {
	if owner == nil {
		return nil
	}
	for index := range owner.Secrets {
		if !owner.Secrets[index].IsRevealed {
			return &owner.Secrets[index]
		}
	}
	return nil
}

func latestMemory(owner *entities.Poble) *entities.Memory {
	if owner == nil || len(owner.Memories) == 0 {
		return nil
	}
	best := &owner.Memories[0]
	for index := range owner.Memories {
		if owner.Memories[index].Timestamp.ToMinutes() > best.Timestamp.ToMinutes() {
			best = &owner.Memories[index]
		}
	}
	return best
}

func iconForItem(item entities.Item) string {
	switch strings.ToLower(item.Type) {
	case "food":
		return "🍎"
	case "tool":
		return "🛠"
	case "medicine":
		return "🩹"
	case "book":
		return "📘"
	default:
		return "•"
	}
}

func physicalDescriptionForItem(item entities.Item) string {
	switch strings.ToLower(item.Type) {
	case "food":
		return "Todavia parece util. No exactamente fresco, pero tampoco una mala idea."
	case "tool":
		return "Tiene marcas de uso reales. Mano, fuerza, rutina."
	case "medicine":
		return "Guardado con mas cuidado que cariño."
	case "book":
		return "Papel cansado, esquinas vencidas, insistencia de quedarse."
	default:
		return "Objeto simple. Textura clara. Nada aqui se ve accidental."
	}
}
