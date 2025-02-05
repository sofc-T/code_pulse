import React from "react";
import ReactDOM from "react-dom";
import styles from "./Modal.module.css";
import { FaTimes } from "react-icons/fa";


export default function Modal({ isOpen, childern, onClose }) {
  if (!isOpen) return null;
  return ReactDOM.createPortal(
    <>
      <div className={styles.overlay} onClick={onClose}/>
      <div className={styles.modal}>
        <button onClick={onClose} className={styles["no-style"]}>
        <FaTimes style={{ cursor: "pointer", color: "red" ,fontSize :"20px" }} />
        </button>
        {childern}
      </div>
    </>,
    document.getElementById("portal")
  );
}
