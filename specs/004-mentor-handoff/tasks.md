# 004. Задачи

- [x] Инструмент `handoff_to_mentor`, получатель – наставник или руководитель.
- [x] Переписка стажера с наставником, переданный вопрос – `kind: handoff`.
- [x] Ответ наставника возвращается в чат AI-buddy; галочка «в справочник» создает статью.
- [x] Согласие стажера перед передачей, правка формулировки, отказ.
- [x] Формулировка вопроса моделью с проверкой результата.
- [x] Защита от «передал» без вызова инструмента.
- [x] Тесты: `internal/app/chat_test.go` (`TestAssistantIntents`, `TestModelClaimsActionWithoutTool`, `TestMentorAnswerVisibility`, `TestChatDialogsHistory`), `internal/app/mentor_test.go` (`TestMentorThreadsAndHandoff`); frontend `tests/mentor.test.tsx`.
- [ ] Напоминание наставнику, если вопрос без ответа дольше дня (ИИ-крон).
