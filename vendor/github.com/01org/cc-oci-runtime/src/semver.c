/*
 * This file is part of cc-oci-runtime.
 *
 * Copyright (C) 2016 Intel Corporation
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
 */

/** \file
 *
 * Semantic Versioning routines.
 *
 * \see http://semver.org/spec/v2.0.0.html
 */
#include <stdlib.h>
#include <stdbool.h>

#include <glib.h>
#include <glib/gprintf.h>

#include "common.h"

/*!
 * Determine if specified string comprises entirely of numeric
 * characters.
 *
 * \param str String to consider.
 *
 * \return \c true on success, else \c false.
 */
private gboolean
cc_oci_string_is_numeric (const char *str)
{
	const gchar *p = str;

	if (! str) {
		return false;
	}

	while (*p) {
		if (! g_ascii_isdigit (*p)) {
			return false;
		}
		p++;
	}

	return true;
}

/*!
 * Compare two pre-release version strings.
 *
 * \param pre_rel_a First pre-release string.
 * \param pre_rel_b Second pre-release string.
 *
 * Precedence for two pre-release versions with the same major,
 * minor, and patch version MUST be determined by comparing each dot
 * separated identifier from left to right until a difference is
 * found as follows:
 *
 * - Identifiers consisting of only digits are compared numerically.
 * - Identifiers with letters or hyphens are compared lexically in ASCII sort order.
 * - Numeric identifiers always have lower precedence than non-numeric identifiers.
 * - A larger set of pre-release fields has a higher precedence than
 *   a smaller set, if all of the preceding identifiers are equal.
 *
 * \return \c strcmp(3)-like values based on whether \p pre_rel_a > \p
 * pre_rel_b.
 */
static gint
cc_oci_semver_cmp_patch_pre_releases (gchar *pre_rel_a,
		gchar *pre_rel_b)
{
	gint        ret = 0;
	guint       i;
	gchar     **fields_a = NULL;
	gchar     **fields_b = NULL;

	g_assert (pre_rel_a);
	g_assert (pre_rel_b);

	fields_a = g_strsplit (pre_rel_a, ".", -1);
	fields_b = g_strsplit (pre_rel_b, ".", -1);

	for (i = 0; ; i++) {
		gchar *fa = fields_a[i];
		gchar *fb = fields_b[i];

		/* A larger set of pre-release fields has a higher
		 * precedence than a smaller set, if all of the
		 * preceding identifiers are equal.
		 */
		if (fa && (!fb)) {
			ret = 1;
			break;
		} else if ((!fa) && fb) {
			ret = -1;
			break;
		} else if (! (fa && fb)) {
			ret = 0;
			break;
		}

		g_assert (fa);
		g_assert (fb);

		if (cc_oci_string_is_numeric (fa)
				&& cc_oci_string_is_numeric (fb)) {
			long la = atol (fa);
			long lb = atol (fb);

			ret = (gint)(la - lb);
		} else {
			ret = g_strcmp0 (fa, fb);
		}

		if (ret) {
			/* fields are different */
			break;
		}
	}

	g_strfreev (fields_a);
	g_strfreev (fields_b);

	return ret;
}

/*!
 * Split a SemVer patch version into its constituent parts.
 *
 * Value patch formats:
 *
 *     <num>
 *     <num>+<build_metadata>
 *     <num>-<pre_release_version>
 *     <num>-<pre_release_version>+<build_metadata>
 *
 * \param patch_version Patch version.
 * \param[out] patch_num Numeric patch number.
 * \param[out] pre_release Pre-release value.
 * \param[out] build_metadata Build metadata value.
 */
static void
cc_oci_semver_split_patch_version (const gchar *patch_version,
		gint *patch_num,
		gchar **pre_release,
		gchar **build_metadata)
{
	g_assert (patch_version);
	g_assert (patch_num);
	g_assert (build_metadata);
	g_assert (pre_release);

	*build_metadata = g_strstr_len (patch_version, -1, "+");
	*pre_release = g_strstr_len (patch_version, -1, "-");

	if (*pre_release) {
		**pre_release = '\0';
		(*pre_release)++;
	}

	if (*build_metadata) {
		**build_metadata= '\0';
		(*build_metadata)++;
	}

	*patch_num = atoi (patch_version);
}

/*!
 * Compare two SemVer patch versions.
 *
 * \param patch_a First patch version.
 * \param patch_b Second patch version.
 *
 * \return \c strcmp(3)-like values based on whether \p patch_a > \p
 * patch_b.
 */
static gint
cc_oci_semver_cmp_patch_versions (const gchar *patch_a,
		const gchar *patch_b)
{
	gint    ret;
	gchar  *a;
	gchar  *b;

	gint    pva = -1;
	gchar  *pra;
	gchar  *bma;

	gint    pvb = -1;
	gchar  *prb;
	gchar  *bmb;

	g_assert (patch_a);
	g_assert (patch_b);

	a = g_strdup (patch_a);
	b = g_strdup (patch_b);

	cc_oci_semver_split_patch_version (a, &pva, &pra, &bma);
	cc_oci_semver_split_patch_version (b, &pvb, &prb, &bmb);

	/* XXX: build metadata does *NOT* form part of the
	 * XXX: precedence calculation!
	 */
	if (pva > pvb) {
		ret = 1;
		goto out;
	} else if (pva < pvb) {
		ret = -1;
		goto out;
	}

	g_assert (pva == pvb);

	/* "a pre-release version has lower precedence than a normal
	 * version".
	 */
	if (pra && (!prb)) {
		ret = -1;
		goto out;
	} else if (prb && (!pra)) {
		ret = 1;
		goto out;
	} else if (! pra && (!prb)) {
		ret = 0;
		goto out;
	}

	g_assert (pra && prb);

	ret = cc_oci_semver_cmp_patch_pre_releases (pra, prb);

out:
	g_free (a);
	g_free (b);

	return ret;
}

/*!
 * Compare two SemVer 2.0.0 strings which have been broken into fields.
 *
 * \param fields_a Fields of first SemVer version string.
 * \param fields_b Fields of second SemVer version string.
 * \param compatible If \c true, compare \ref fields_a and \ref fields_b
 *   for SemVer backwards-compatible differences only. This is a less
 *   restrictive comparison that only checks the major numbers.
 *
 * \return \c strcmp(3)-like values based on whether \p fields_a > \p
 * fields_b.
 */
static gint
cc_oci_semver_cmp_fields (const gchar **fields_a,
		const gchar **fields_b,
		gboolean compatible)
{
	glong maj_a, min_a;
	glong maj_b, min_b;

	const gchar *major_a = fields_a[0];
	const gchar *minor_a = fields_a[1];
	const gchar *patch_a = fields_a[2];

	const gchar *major_b = fields_b[0];
	const gchar *minor_b = fields_b[1];
	const gchar *patch_b = fields_b[2];

	g_assert (fields_a);
	g_assert (fields_b);

	maj_a = atol (major_a);
	min_a = atol (minor_a);

	maj_b = atol (major_b);
	min_b = atol (minor_b);

	if (maj_a > maj_b) {
		return 1;
	} else if (maj_a < maj_b) {
		return -1;
	} else if (compatible) {

		/* Major numbers are identical so SemVer mandates that
		 * all other fields must refer to backwards compatible
		 * changes.
		 */
		return 0;
	}

	g_assert (maj_a == maj_b);

	if (min_a > min_b) {
		return 1;
	} else if (min_a < min_b) {
		return -1;
	}

	g_assert (min_a == min_b);

	return cc_oci_semver_cmp_patch_versions (patch_a, patch_b);
}

/*!
 * Compare two SemVer 2.0.0 strings.
 *
 * \param version_a First SemVer string.
 * \param version_b Second SemVer string.
 * \param compatible If \c true \ref version_a and \ref version_b are
 *   compared for backwards-compatibility differences only.
 *
 * \return \c 0 if \p version_a == \p version_b, \c <0 if \p version_a <
 * \p version_b or \c >0 if \p version_a > \p version_b.
 */
static gint
cc_oci_semver_2_0_0_cmp (const char *version_a, const char *version_b,
		gboolean compatible)
{
	gint ret;
	gchar **fields_a = NULL;
	gchar **fields_b = NULL;

	g_assert (version_a);
	g_assert (version_b);

	/* The "3" arg to g_strsplit() is to ensure the strings are
	 * split into exactly these fields:
	 *
	 *  - major
	 *  - minor
	 *  - patch-level
	 *
	 * We don't want to split the entire string since patch-level
	 * may contain periods itself, but that is handled later.
	 */
	fields_a = g_strsplit (version_a, ".", 3);
	fields_b = g_strsplit (version_b, ".", 3);

	g_assert (fields_a);
	g_assert (fields_b);

	ret = cc_oci_semver_cmp_fields ((const gchar **)fields_a,
			(const gchar **)fields_b, compatible);

	g_strfreev (fields_a);
	g_strfreev (fields_b);

	return ret;
}

/*!
 * Compare two Semantic version (SemVer) strings for
 * backwards-compatibility.
 *
 * \param version_a First SemVer string.
 * \param version_b Second SemVer string.
 *
 * Return values match those of \c strcmp(3).
 *
 * \note Handles SemVer 2.0.0-format strings.
 *
 * \return \c 0 if \p version_a == \p version_b, \c <0 if \p version_a <
 * \p version_b or \c >0 if \p version_a > \p version_b.
 */
gint
cc_oci_semver_cmp (const char *version_a, const char *version_b)
{
	return cc_oci_semver_2_0_0_cmp (version_a, version_b, true);
}
