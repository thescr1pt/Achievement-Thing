package toast

import (
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
)

const (
	Default = "ms-winsoundevent:Notification.Default"
	Silent  = "silent"
)

const (
	ToastGeneric = "ToastGeneric"
)

type toastXML struct {
	XMLName xml.Name `xml:"toast"`
	Visual  visual   `xml:"visual"`
	Audio   audio    `xml:"audio"`
}

type visual struct {
	Binding binding `xml:"binding"`
}

type binding struct {
	Template string        `xml:"template,attr"`
	Texts    []textElement `xml:"text"`
	Image    *image        `xml:"image,omitempty"`
}

type textElement struct {
	Placement string `xml:"placement,attr,omitempty"`
	Value     string `xml:",chardata"`
}

type image struct {
	Src       string `xml:"src,attr"`
	Placement string `xml:"placement,attr"`
}

type audio struct {
	Src string `xml:"src,attr"`
}

type Toast struct {
	AppID       string
	Icon        string
	Title       string
	Message     string
	Audio       string
	Attribution string
}

func (t *Toast) Show() error {
	xmlContent, err := t.buildXML()
	if err != nil {
		return fmt.Errorf("failed to build XML: %w", err)
	}

	if err := invoke(xmlContent, t.AppID); err != nil {
		return fmt.Errorf("failed to show toast notification: %w", err)
	}

	return nil
}

func (t *Toast) buildXML() (string, error) {
	texts := []textElement{
		{Value: t.Title},
		{Value: t.Message},
	}

	if strings.TrimSpace(t.Attribution) != "" {
		texts = append(texts, textElement{
			Placement: "attribution",
			Value:     t.Attribution,
		})
	}

	binding := binding{
		Template: ToastGeneric,
		Texts:    texts,
	}

	if strings.TrimSpace(t.Icon) != "" {
		binding.Image = &image{
			Src:       t.Icon,
			Placement: "appLogoOverride",
		}
	}

	toastXML := toastXML{
		Visual: visual{Binding: binding},
		Audio:  audio{Src: t.Audio},
	}

	xmlBytes, err := xml.Marshal(toastXML)
	if err != nil {
		return "", fmt.Errorf("failed to marshal XML: %w", err)
	}

	return string(xmlBytes), nil
}

func invoke(xmlContent, appID string) error {
	if strings.TrimSpace(xmlContent) == "" {
		return fmt.Errorf("xmlContent cannot be empty")
	}
	if strings.TrimSpace(appID) == "" {
		return fmt.Errorf("appID cannot be empty")
	}

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command",
		`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null;
		[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null;
			$xml = New-Object Windows.Data.Xml.Dom.XmlDocument;
			$xml.LoadXml('`+xmlContent+`');
			$toast = New-Object Windows.UI.Notifications.ToastNotification($xml);
			[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('`+appID+`').Show($toast)`)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("PowerShell execution failed: %w", err)
	}

	return nil
}
