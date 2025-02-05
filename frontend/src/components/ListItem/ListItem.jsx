import { FaTrash } from "react-icons/fa";
import styles from "./ListItem.module.css";

export default function ListItem({ user, onClick }) {
  return (
    <li className={styles.li}>
      <p>{user}</p>
      {user === "You" ? null : <FaTrash onClick={() => onClick(user)} style={{ cursor: "pointer", color: "red" }} />}
    </li>
  );
}
