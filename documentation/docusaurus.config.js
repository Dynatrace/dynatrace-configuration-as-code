/** @type {import('@docusaurus/types').DocusaurusConfig} */

const versions = require("./versions.json");
const latestVersion = versions[0];

module.exports = {
  title: 'Monaco',
  tagline: 'Automate Monitoring',
  url: 'https://dynatrace-oss.github.io',
  baseUrl: '/dynatrace-monitoring-as-code/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  favicon: 'img/favicon.png',
  organizationName: 'dynatrace-oss',
  projectName: 'dynatrace-monitoring-as-code',
  themeConfig: {
    colorMode: {
      defaultMode: 'dark',
    },
    navbar: {
      title: 'Monaco',
      logo: {
        alt: 'My Site Logo',
        src: 'img/DT_Logo.svg',
      },
      items: [
        {
          type: 'doc',
          docId: 'intro',
          position: 'left',
          label: 'Docs',
        },
        {
          type: "docsVersionDropdown",
          position: "right",
          dropdownActiveClassDisabled: true,
          dropdownItemsAfter: [
            {
              to: "/versions",
              label: "All versions",
            },
          ],
        },
        {
          href: 'https://github.com/dynatrace-oss/dynatrace-monitoring-as-code',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      copyright: `Monaco ${new Date().getFullYear()}.`,
    },
  },
  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl: 'https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/edit/main/documentation/',
          versions: {},
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],
  plugins: [
    [require.resolve('@cmfcmf/docusaurus-search-local'), {
      indexDocs: true,
      language: "en",
      docsRouteBasePath: '/',
    }]
  ],
};
