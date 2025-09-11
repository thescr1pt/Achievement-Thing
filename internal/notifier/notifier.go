package notifier

import (
	"Achievement-Thing/pkg/toast"
	"fmt"
)

func SendAchievement(title, message, icon string) error {
	fmt.Println("Sending achievement notification:", title)
	notification := toast.Toast{
		AppID:   "Microsoft.XboxGamingOverlay_8wekyb3d8bbwe!App",
		Title:   title,
		Message: message,
		Icon:    icon,
		Audio:   "ms-winsoundevent:Notification.AchievementThing",
	}

	return notification.Show()
}
