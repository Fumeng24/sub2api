-- Upstream 0435417f43 added this as migration 158. Local migration numbers
-- already passed that range, so keep the SQL identical and append it as 174.
--
-- PR 3593 added Grok media routes for image generation, image edits, and video generation.
-- Existing Grok groups were created before the image-generation gate knew about
-- the Grok platform, so backfill them onto the same generation capability gate.
UPDATE groups
SET allow_image_generation = true
WHERE platform = 'grok'
  AND allow_image_generation = false;
