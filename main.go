package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Contact represents a single contact in the phone book
// It stores the name and phone number of a contact with database ID
type Contact struct {
	ID    int64 // Database ID for the contact
	Name  string
	Phone string
}

// PhoneBookApp represents our main application structure
// It holds all the UI components, data, and database connection
type PhoneBookApp struct {
	// Core application window
	window fyne.Window

	// Database connection
	db *sql.DB

	// Data storage
	contacts         []Contact // Slice to store all contacts
	filteredContacts []Contact // Slice for search results

	// UI Components
	contactList *widget.List  // List widget to display contacts
	nameEntry   *widget.Entry // Input field for contact name
	phoneEntry  *widget.Entry // Input field for phone number
	searchEntry *widget.Entry // Search input field
	statusLabel *widget.Label // Status bar label

	// State management
	currentContact int // Index of currently selected contact (-1 if none)
}

func main() {
	// Create a new Fyne application
	myApp := app.New()

	// Create the main window with a title
	myWindow := myApp.NewWindow("Phone Book Manager")

	// Set a reasonable default size for the window
	myWindow.Resize(fyne.NewSize(600, 500))

	// Initialize database
	db, err := initDatabase()
	if err != nil {
		// Show error dialog if database fails to initialize
		dialog.ShowError(fmt.Errorf("Failed to initialize database: %v", err), myWindow)
		log.Fatal(err) // Log fatal error
	}
	defer db.Close() // Ensure database connection is closed when app exits

	// Load contacts from database
	contacts, err := loadContacts(db)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to load contacts: %v", err), myWindow)
		log.Printf("Warning: Could not load contacts: %v", err)
		contacts = make([]Contact, 0) // Start with empty list if load fails
	}

	// Initialize our phone book application
	phoneBook := &PhoneBookApp{
		window:           myWindow,
		db:               db,
		contacts:         contacts,
		filteredContacts: make([]Contact, len(contacts)),
		currentContact:   -1, // No contact selected
	}

	// Copy contacts to filtered list
	copy(phoneBook.filteredContacts, contacts)

	// Build the user interface
	phoneBook.buildUI()

	// Show the window and start the application
	myWindow.ShowAndRun()
}

// initDatabase creates the database file and contacts table if they don't exist
func initDatabase() (*sql.DB, error) {
	// Open SQLite database (creates file if it doesn't exist)
	db, err := sql.Open("sqlite3", "./phonebook.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create contacts table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		phone TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	return db, nil
}

// loadContacts retrieves all contacts from the database
func loadContacts(db *sql.DB) ([]Contact, error) {
	rows, err := db.Query("SELECT id, name, phone FROM contacts ORDER BY name COLLATE NOCASE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		err := rows.Scan(&c.ID, &c.Name, &c.Phone)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return contacts, nil
}

// saveContact inserts a new contact into the database
func (app *PhoneBookApp) saveContact(name, phone string) (int64, error) {
	result, err := app.db.Exec(
		"INSERT INTO contacts (name, phone) VALUES (?, ?)",
		name, phone,
	)
	if err != nil {
		return 0, err
	}

	// Get the ID of the newly inserted contact
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// updateContactInDB updates an existing contact in the database
func (app *PhoneBookApp) updateContactInDB(id int64, name, phone string) error {
	_, err := app.db.Exec(
		"UPDATE contacts SET name = ?, phone = ? WHERE id = ?",
		name, phone, id,
	)
	return err
}

// deleteContactFromDB removes a contact from the database
func (app *PhoneBookApp) deleteContactFromDB(id int64) error {
	_, err := app.db.Exec("DELETE FROM contacts WHERE id = ?", id)
	return err
}

// buildUI creates all the user interface components and arranges them
func (app *PhoneBookApp) buildUI() {
	// ==================== INPUT SECTION ====================
	// Create input fields for adding/editing contacts
	app.nameEntry = widget.NewEntry()
	app.nameEntry.SetPlaceHolder("Enter contact name...")

	app.phoneEntry = widget.NewEntry()
	app.phoneEntry.SetPlaceHolder("Enter phone number...")

	// Create buttons for contact operations
	addButton := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		app.addContact()
	})
	addButton.Importance = widget.HighImportance

	updateButton := widget.NewButtonWithIcon("Update", theme.DocumentSaveIcon(), func() {
		app.updateContact()
	})

	deleteButton := widget.NewButtonWithIcon("Delete", theme.ContentRemoveIcon(), func() {
		app.deleteContact()
	})
	deleteButton.Importance = widget.DangerImportance

	clearButton := widget.NewButtonWithIcon("Clear", theme.CancelIcon(), func() {
		app.clearForm()
	})

	// Arrange input fields and buttons in a form
	inputForm := container.NewVBox(
		widget.NewLabelWithStyle("Contact Details", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		app.nameEntry,
		app.phoneEntry,
		container.NewGridWithColumns(4,
			addButton,
			updateButton,
			deleteButton,
			clearButton,
		),
	)

	// ==================== SEARCH SECTION ====================
	// Create search functionality
	app.searchEntry = widget.NewEntry()
	app.searchEntry.SetPlaceHolder("Search contacts...")
	app.searchEntry.OnChanged = func(s string) {
		app.filterContacts(s) // Filter contacts as user types
	}

	searchBox := container.NewBorder(
		nil, nil, nil, nil,
		container.NewHBox(
			widget.NewIcon(theme.SearchIcon()),
			app.searchEntry,
		),
	)

	// ==================== CONTACT LIST SECTION ====================
	// Create the contact list widget
	app.contactList = widget.NewList(
		// Return the number of items in the list
		func() int {
			return len(app.filteredContacts)
		},
		// Create a new template item for the list
		func() fyne.CanvasObject {
			// This defines how each contact will look
			return container.NewHBox(
				widget.NewIcon(theme.AccountIcon()),
				widget.NewLabel("Template"), // Will be replaced with actual name
				widget.NewLabel(""),         // Will be replaced with phone number
			)
		},
		// Update the template with actual contact data
		func(id widget.ListItemID, item fyne.CanvasObject) {
			// Get the contact at this position
			contact := app.filteredContacts[id]

			// Update the labels with contact information
			box := item.(*fyne.Container)
			nameLabel := box.Objects[1].(*widget.Label)
			phoneLabel := box.Objects[2].(*widget.Label)

			nameLabel.SetText(contact.Name)
			phoneLabel.SetText(contact.Phone)
		},
	)

	// Handle contact selection
	app.contactList.OnSelected = func(id widget.ListItemID) {
		// Find the actual contact index in the full list
		selectedContact := app.filteredContacts[id]
		for i, contact := range app.contacts {
			if contact.ID == selectedContact.ID { // Compare by ID instead of name/phone
				app.currentContact = i
				break
			}
		}

		// Fill the form with selected contact's details
		app.nameEntry.SetText(selectedContact.Name)
		app.phoneEntry.SetText(selectedContact.Phone)
	}

	// ==================== STATUS SECTION ====================
	// Create status label
	app.statusLabel = widget.NewLabel("Ready")
	app.updateStatus()

	// ==================== MAIN LAYOUT ====================
	// Arrange all sections in the main window
	content := container.NewBorder(
		// Top section: Input form
		container.NewVBox(
			inputForm,
			widget.NewSeparator(),
			searchBox,
		),
		// Bottom section: Status bar
		app.statusLabel,
		// Left/Right: None
		nil,
		nil,
		// Center section: Contact list
		container.NewBorder(
			widget.NewLabelWithStyle("Contacts", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			nil, nil, nil,
			app.contactList,
		),
	)

	app.window.SetContent(content)
}

// addContact adds a new contact to the phone book and database
func (app *PhoneBookApp) addContact() {
	// Validate input
	name := strings.TrimSpace(app.nameEntry.Text)
	phone := strings.TrimSpace(app.phoneEntry.Text)

	if name == "" || phone == "" {
		dialog.ShowInformation("Error", "Please enter both name and phone number", app.window)
		return
	}

	// Check for duplicate names (optional)
	for _, contact := range app.contacts {
		if strings.EqualFold(contact.Name, name) {
			dialog.ShowConfirm("Duplicate",
				"A contact with this name already exists. Add anyway?",
				func(proceed bool) {
					if proceed {
						app.saveNewContact(name, phone)
					}
				},
				app.window)
			return
		}
	}

	// Save to database
	app.saveNewContact(name, phone)
}

// saveNewContact handles the actual database save operation
func (app *PhoneBookApp) saveNewContact(name, phone string) {
	// Save to database
	id, err := app.saveContact(name, phone)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to save contact: %v", err), app.window)
		return
	}

	// Add to local slice
	app.contacts = append(app.contacts, Contact{ID: id, Name: name, Phone: phone})
	app.afterContactChange()

	// Clear form and show success message
	app.clearForm()
	dialog.ShowInformation("Success", "Contact added successfully!", app.window)
}

// updateContact updates the currently selected contact
func (app *PhoneBookApp) updateContact() {
	// Check if a contact is selected
	if app.currentContact < 0 || app.currentContact >= len(app.contacts) {
		dialog.ShowInformation("Error", "Please select a contact to update", app.window)
		return
	}

	// Validate input
	name := strings.TrimSpace(app.nameEntry.Text)
	phone := strings.TrimSpace(app.phoneEntry.Text)

	if name == "" || phone == "" {
		dialog.ShowInformation("Error", "Please enter both name and phone number", app.window)
		return
	}

	// Get the contact to update
	contact := app.contacts[app.currentContact]

	// Update in database
	err := app.updateContactInDB(contact.ID, name, phone)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to update contact: %v", err), app.window)
		return
	}

	// Update in local slice
	app.contacts[app.currentContact] = Contact{ID: contact.ID, Name: name, Phone: phone}
	app.afterContactChange()

	dialog.ShowInformation("Success", "Contact updated successfully!", app.window)
}

// deleteContact removes the currently selected contact
func (app *PhoneBookApp) deleteContact() {
	// Check if a contact is selected
	if app.currentContact < 0 || app.currentContact >= len(app.contacts) {
		dialog.ShowInformation("Error", "Please select a contact to delete", app.window)
		return
	}

	contact := app.contacts[app.currentContact]

	// Show confirmation dialog
	dialog.ShowConfirm("Delete Contact",
		fmt.Sprintf("Are you sure you want to delete %s?", contact.Name),
		func(proceed bool) {
			if proceed {
				// Delete from database
				err := app.deleteContactFromDB(contact.ID)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to delete contact: %v", err), app.window)
					return
				}

				// Remove from local slice
				app.contacts = append(app.contacts[:app.currentContact], app.contacts[app.currentContact+1:]...)
				app.currentContact = -1
				app.afterContactChange()
				app.clearForm()

				dialog.ShowInformation("Success", "Contact deleted successfully!", app.window)
			}
		},
		app.window)
}

// clearForm resets all input fields
func (app *PhoneBookApp) clearForm() {
	app.nameEntry.SetText("")
	app.phoneEntry.SetText("")
	app.contactList.UnselectAll()
	app.currentContact = -1
}

// filterContacts filters the contact list based on search text
func (app *PhoneBookApp) filterContacts(searchText string) {
	// If search is empty, show all contacts
	if searchText == "" {
		app.filteredContacts = make([]Contact, len(app.contacts))
		copy(app.filteredContacts, app.contacts)
	} else {
		// Filter contacts that contain the search text (case insensitive)
		searchLower := strings.ToLower(searchText)
		app.filteredContacts = make([]Contact, 0)

		for _, contact := range app.contacts {
			if strings.Contains(strings.ToLower(contact.Name), searchLower) ||
				strings.Contains(contact.Phone, searchText) {
				app.filteredContacts = append(app.filteredContacts, contact)
			}
		}
	}

	// Refresh the list to show filtered results
	if app.contactList != nil {
		app.contactList.Refresh()
	}
	app.updateStatus()
}

// afterContactChange is called after any contact list modification
func (app *PhoneBookApp) afterContactChange() {
	// Sort contacts alphabetically
	sort.Slice(app.contacts, func(i, j int) bool {
		return strings.ToLower(app.contacts[i].Name) < strings.ToLower(app.contacts[j].Name)
	})

	// Update filtered list and refresh
	if app.searchEntry != nil {
		app.filterContacts(app.searchEntry.Text)
	} else {
		app.filteredContacts = make([]Contact, len(app.contacts))
		copy(app.filteredContacts, app.contacts)
	}

	app.updateStatus()
}

// updateStatus updates the status bar with contact count
func (app *PhoneBookApp) updateStatus() {
	if app.statusLabel != nil {
		total := len(app.contacts)
		shown := len(app.filteredContacts)
		if total == shown {
			app.statusLabel.SetText(fmt.Sprintf("Total contacts: %d", total))
		} else {
			app.statusLabel.SetText(fmt.Sprintf("Showing %d of %d contacts", shown, total))
		}
	}
}
