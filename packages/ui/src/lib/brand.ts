// Static brand facts the design needs as text. Club data that changes (stats, terms, events) comes from the
// snapshot; these are the words and links the club uses everywhere.
export const BRAND = {
  name: "Binary SUST",
  bangla: "বাইনারি",
  /** The line under the logo. */
  motto: "technology for real life",
  /** The club's own mission statement; quote it verbatim. */
  mission: "Aiming to boost digital skills, foster teamwork, and inspire creativity among members.",
  /** Objectives from the club's constitution. */
  objectives: [
    "Increase computer skills and teamwork skills",
    "Build skills in computing and in presenting ideas",
    "Share technological information and analysis",
    "Promote creativity",
  ],
  university: "Shahjalal University of Science and Technology, Sylhet",
  email: "binary.sust@gmail.com",
  links: [
    { label: "Facebook", href: "https://www.facebook.com/binary.sust" },
    { label: "LinkedIn", href: "https://www.linkedin.com/company/binary-sust" },
  ],
  /**
   * The official logo, a path each app serves from its own public/ (apps/web/public/brand/logo.png). Set it to
   * "/brand/logo.svg" once that file exists; null makes <Logo> set the name in type.
   */
  logoSrc: "/brand/logo.png" as string | null,
} as const;
