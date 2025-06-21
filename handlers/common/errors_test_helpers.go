package common

import "oba-twilio/models"

// Test helpers to expose private methods for testing

func (e *ErrorHandler) MapAppErrorToUserMessage(appErr *models.AppError, language string) (string, []interface{}) {
	return e.mapAppErrorToUserMessage(appErr, language)
}

func (e *ErrorHandler) GetLocalizedErrorMessage(err error, language, channel string) string {
	return e.getLocalizedErrorMessage(err, language, channel)
}

func (e *ErrorHandler) LogError(context string, err error) {
	e.logError(context, err)
}
