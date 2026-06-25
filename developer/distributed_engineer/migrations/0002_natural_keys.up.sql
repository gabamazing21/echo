-- Stable natural keys so the seeder can UPSERT content on every boot without
-- ever deleting rows — which means your progress (FK'd to these ids) survives
-- content updates. (step_id, position) uniquely identifies a seeded row.
ALTER TABLE tasks     ADD CONSTRAINT uq_tasks_step_pos     UNIQUE (step_id, position);
ALTER TABLE resources ADD CONSTRAINT uq_resources_step_pos UNIQUE (step_id, position);
ALTER TABLE quizzes   ADD CONSTRAINT uq_quizzes_step_pos   UNIQUE (step_id, position);
