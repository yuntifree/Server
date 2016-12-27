DROP PROCEDURE IF EXISTS yunxing.purchase_sales;
delimiter //
CREATE PROCEDURE yunxing.purchase_sales ( IN _sid BIGINT UNSIGNED, IN _uid INT UNSIGNED,
	IN _price INT UNSIGNED)
BEGIN
	DECLARE user_balance INT UNSIGNED default 0;
	DECLARE remain_times INT UNSIGNED default 0;
	DECLARE ret	INT UNSIGNED default 0;
	DECLARE sales_hid INT UNSIGNED default 0;
	START TRANSACTION;
	SELECT balance FROM user WHERE uid = _uid INTO user_balance FOR UPDATE;
	IF user_balance >= _price THEN
		SELECT hid FROM sales_history WHERE sid = _sid AND uid = 0 ORDER BY RAND() LIMIT 1 INTO sales_hid FOR UPDATE;
		SELECT remain FROM sales WHERE sid = _sid INTO remain_times FOR UPDATE;
		IF remain_times >= 1 THEN
			UPDATE sales SET remain = remain - 1 WHERE sid = _sid;
			UPDATE sales_history SET uid = _uid, ctime = NOW() WHERE hid = sales_hid;
			UPDATE user SET balance = balance - _price WHERE uid = _uid;
		ELSE 
			SET ret = 3;
		END IF;
	ELSE 
		SET ret = 2;
	END IF;
	COMMIT;
	SELECT ret, sales_hid;
END//
