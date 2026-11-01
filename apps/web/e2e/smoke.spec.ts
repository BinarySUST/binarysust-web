import { expect, test } from "@playwright/test";
import { mockApi, slice } from "./api";

type PersonCard = { handle: string; name: string };
type EventCard = {
  slug: string;
  title: string;
  status: string;
  endsAt: string;
  registration: { opensAt: string; closesAt: string } | null;
};

test.beforeEach(async ({ page }) => {
  await mockApi(page);
});

test("the landing page renders from the snapshot", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle(/Binary SUST/);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Main" }).first()).toBeVisible();
});

test("people directory links to a portfolio", async ({ page }) => {
  const [first] = slice<PersonCard[]>("people");
  expect(first, "the snapshot lists at least one person").toBeDefined();
  await page.goto("/people");
  await page.locator(`main a[href="/people/${first!.handle}"]`).first().click();
  await expect(page).toHaveURL(new RegExp(`/people/${first!.handle}$`));
  await expect(page.getByRole("heading", { level: 1, name: first!.name })).toBeVisible();
});

test("an event with registration shows the RSVP call to action", async ({ page }) => {
  const event = slice<EventCard[]>("events").find(
    (e) => e.status === "published" && e.registration && Date.parse(e.registration.opensAt) < Date.parse(e.endsAt),
  );
  expect(event, "the snapshot has an event with registration").toBeDefined();
  // Inside the registration window, whatever today is.
  await page.clock.setFixedTime(new Date(Date.parse(event!.registration!.opensAt) + 60_000));
  await page.goto(`/events/${event!.slug}`);
  await expect(page.getByRole("heading", { level: 1, name: event!.title })).toBeVisible();
  const cta = page.getByRole("button", { name: "Sign in to reserve" });
  await expect(cta).toBeVisible();
  // The button is in the pre-rendered HTML; it works once React has hydrated the island.
  await expect(async () => {
    await cta.click();
    await expect(page).toHaveURL(/\/login\?next=/, { timeout: 1_000 });
  }).toPass();
  expect(new URL(page.url()).searchParams.get("next")).toBe(`/events/${event!.slug}`);
});

test("the dashboard sends a visitor to sign in", async ({ page }) => {
  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/login\?next=%2Fdashboard/);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
});

test("the dashboard shows only the sections the account may use", async ({ page }) => {
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await mockApi(page, { permissions: ["profile.edit_own", "blog.write", "events.manage"], roles: ["event_manager"] });
  await page.goto("/dashboard");
  const nav = page.getByRole("navigation", { name: "Dashboard" });
  await expect(nav.getByRole("link", { name: "Overview" })).toBeVisible();
  for (const name of ["My portfolio", "My RSVPs", "Events", "Blog"]) {
    await expect(nav.getByRole("link", { name, exact: true })).toBeVisible();
  }
  for (const name of ["People", "Committee & roles", "Talks", "Sponsors", "Media"]) {
    await expect(nav.getByRole("link", { name, exact: true })).toHaveCount(0);
  }
  await expect(nav).toContainText("Test Member");
});
