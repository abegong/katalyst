package checks

// CheckType is the `kind:` identifier selecting a check type. It lives here, in
// the package that owns the check registry, so check types and their parsers no
// longer depend on internal/project/config for their identity.
type CheckType string

const (
	CheckObject                        CheckType = "object"
	CheckObjectRequiredField           CheckType = "object_required_field"
	CheckObjectFieldType               CheckType = "object_field_type"
	CheckObjectFieldEnum               CheckType = "object_field_enum"
	CheckObjectNumberRange             CheckType = "object_number_range"
	CheckObjectStringLength            CheckType = "object_string_length"
	CheckObjectSentenceCase            CheckType = "object_sentence_case"
	CheckMarkdownTitleMatchesH1        CheckType = "markdown_title_matches_h1"
	CheckMarkdownRequiresH1            CheckType = "markdown_requires_h1"
	CheckMarkdownSingleH1              CheckType = "markdown_single_h1"
	CheckMarkdownNoHeadingLevelJumps   CheckType = "markdown_no_heading_level_jumps"
	CheckMarkdownRequiredSection       CheckType = "markdown_required_section"
	CheckMarkdownCodeFenceHasLanguage  CheckType = "markdown_code_fence_language_required"
	CheckMarkdownWritingTells          CheckType = "markdown_writing_tells"
	CheckFilesystemExtensionIn         CheckType = "filesystem_extension_in"
	CheckFilesystemParentDirIn         CheckType = "filesystem_parent_dir_in"
	CheckFilesystemNameCase            CheckType = "filesystem_name_case"
	CheckFilesystemNameMatchesField    CheckType = "filesystem_name_matches_field"
	CheckFilesystemNameAffix           CheckType = "filesystem_name_affix"
	CheckFilesystemPathCharset         CheckType = "filesystem_path_charset"
	CheckFilesystemNameRegex           CheckType = "filesystem_name_regex"
	CheckFilesystemNameLength          CheckType = "filesystem_name_length"
	CheckFilesystemPathDepth           CheckType = "filesystem_path_depth"
	CheckFilesystemParentDirMatchesFld CheckType = "filesystem_parent_dir_matches_field"
	CheckFilesystemReferencedFiles     CheckType = "filesystem_referenced_files_exist"
	CheckFilesystemUniqueFilename      CheckType = "filesystem_unique_filename"
	CheckFilesystemUniqueField         CheckType = "filesystem_unique_field"
	CheckFilesystemIndexFileRequired   CheckType = "filesystem_index_file_required"
	CheckTextRequires                  CheckType = "text_requires"
	CheckTextForbids                   CheckType = "text_forbids"
	CheckTextDenylist                  CheckType = "text_denylist"
)
