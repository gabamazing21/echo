ALTER TABLE quizzes   DROP CONSTRAINT IF EXISTS uq_quizzes_step_pos;
ALTER TABLE resources DROP CONSTRAINT IF EXISTS uq_resources_step_pos;
ALTER TABLE tasks     DROP CONSTRAINT IF EXISTS uq_tasks_step_pos;
