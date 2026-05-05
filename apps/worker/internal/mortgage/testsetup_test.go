package mortgage

import (
	"github.com/mrsimonemms/mortgage-application/apps/worker/internal/mortgage/activities"
)

func init() {
	activities.DisableActivityDelaysForTests()
}
