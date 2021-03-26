package chat

import "strings"

const (
	doReply         = "e:reply"
	doReplyAt       = "e:replyAt"
	doAcceptRequest = "e:acceptReq"
	doRejectRequest = "e:rejectReq"
	doPostMore      = "e:postMore"
	doPostBack      = "e:postBack"
	doTrackPost     = "e:trackPost"
	doUntrackPost   = "e:untrackPost"

	doTrackPostByURL   = "trackPostByURL"
	doUntrackPostByURL = "untrackPostByURL"
)

func isEventAction(action string) bool {
	return strings.HasPrefix(action, "e:")
}
